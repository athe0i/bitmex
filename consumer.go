package main

import (
	"bitmex/bitmex"
	"context"
	"sync"
)

// Intended to receive an update from the Broadcaster and push it
// down the stream to the socket.
type Consumer struct {
	updQueue []bitmex.Update
	acceptedSymbols []string
	signal chan int
	data chan bitmex.Update

	// guard our updQueue
	mu sync.Mutex
}

func NewConsumer(data chan bitmex.Update) (cons *Consumer) {
	return &Consumer{
		updQueue:        make([]bitmex.Update, 0),
		acceptedSymbols: make([]string, 0),
		signal:          make(chan int),
		data:            data,
		mu:              sync.Mutex{},
	}
}

// Might be overcomplication, but idea is to not block Broadcaster
// to wait until upd would be consumed by socket, so we are
// using intermediate queue, something similar was used for "tickle pattern"
// for unblocking concurrency as far as i've got it
func (c *Consumer) Push(upd bitmex.Update) {
	if c.isUpdateAccepted(upd) == false {
		return
	}

	defer c.tickle()

	c.mu.Lock()
	c.updQueue = append(c.updQueue, upd)
	c.mu.Unlock() // if we defer this - we are locking out queue push part

	// "tickle" our Consumer for saying that something had came up
}

func (c *Consumer) tickle() {
	c.signal <- 1
}

// listen for a signals that new upd arrived and push queue down the road
func (c *Consumer) Consume(ctx context.Context) {
	for {
		select {
			case <-c.signal:
				c.pushQueue()
				continue
			case <-ctx.Done():
				return
			}
	}
}

// Filtering out clients updates
func (c *Consumer) isUpdateAccepted(upd bitmex.Update) bool {
	if len(c.acceptedSymbols) == 0 {
		return true
	}

	for _, accepted := range c.acceptedSymbols {
		if upd.Symbol == accepted {
			return true
		}
	}

	return false
}

// push the queue to the channel
func (c *Consumer) pushQueue() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, upd := range c.updQueue {
		c.data <- upd
	}
	// just cleanup all of the updates
	c.updQueue = make([]bitmex.Update, 0)
}

// set filter for our updates
func (c *Consumer) SetAcceptedSymbols(symbols []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.acceptedSymbols = symbols
}