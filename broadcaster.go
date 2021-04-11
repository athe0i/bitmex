package main

import (
	"bitmex/bitmex"
	"context"
	"sync"
)

// Broadcaster listens channel of updates from the Bitmex client
// then pushes the updates down to the client consumers
type Broadcaster struct {
	consumers []*Consumer

	mu sync.Mutex
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		consumers: make([]*Consumer, 0),
		mu:        sync.Mutex{},
	}
}

func (b *Broadcaster) ListenForUpdates(ctx context.Context, data chan bitmex.Update) {
	for {
		select {
		case upd := <-data:
			b.broadcast(upd)
		case <-ctx.Done():
			return
		}
	}
}

func (b *Broadcaster) AddConsumer(cons *Consumer) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// do not double up consumers
	for _, regCons := range b.consumers {
		if regCons == cons {
			return
		}
	}

	b.consumers = append(b.consumers, cons)
}

func (b *Broadcaster) RemoveConsumer(cons *Consumer) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ind := -1

	for i, registeredCons := range b.consumers {
		if registeredCons == cons {
			ind = i
			break
		}
	}

	if ind != -1 {
		b.consumers = append(b.consumers[:ind], b.consumers[ind+1:]...)
	}
}

func (b *Broadcaster) broadcast(upd bitmex.Update) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, cons := range b.consumers {
		cons.Push(upd)
	}
}


