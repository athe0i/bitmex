package main

import (
	"bitmex/bitmex"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)


const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WsClient struct {
	con *Consumer
	broadcst *Broadcaster

	data chan bitmex.Update
	ws *websocket.Conn
}

func NewWsClient(w http.ResponseWriter, r *http.Request, broadcst *Broadcaster) (wsc *WsClient, err error) {
	data := make(chan bitmex.Update)
	con := NewConsumer(data)

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err.Error())
		return
	}

	return &WsClient{
		con:      con,
		broadcst: broadcst,
		ws:       ws,
		data:     data,
	}, err
}

func (wsc *WsClient) Serve(ctx context.Context) {
	//fmt.Println("serving new client")
	wsc.broadcst.AddConsumer(wsc.con)
	defer wsc.Close()

	go wsc.readMessages(ctx)
	go wsc.pushUpdates(ctx)

	select {
	case <-ctx.Done():
		return
	}
}

func (wsc *WsClient) readMessages(ctx context.Context) {
	wsc.ws.SetReadLimit(maxMessageSize)

	for {
		_, msg, err := wsc.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}

		// again, this should be mapped to some action struct.... but it won't
		msgMap := make(map[string]interface{})
		err = json.Unmarshal(msg, &msgMap)
		if err != nil {
			fmt.Println(err)
		}
		wsc.handleMessage(msgMap)

		select {
		case <-ctx.Done():
			return
		default:
			continue
		}
	}
}

func (wsc *WsClient) pushUpdates(ctx context.Context) {
	go wsc.con.Consume(ctx)

	for {
		select {
		case upd := <-wsc.data:
			err := wsc.writeUpdate(upd)
			if err != nil {
				fmt.Println(err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (wsc *WsClient) Close() {
	wsc.broadcst.RemoveConsumer(wsc.con)
	wsc.ws.Close()
}

func (wsc *WsClient) writeUpdate(upd bitmex.Update) error {
	updBytes, _ := json.Marshal(upd)

	return wsc.write(websocket.TextMessage, updBytes)
}

func (wsc *WsClient) write(mt int, payload []byte) error {
	err := wsc.ws.SetWriteDeadline(time.Now().Add(writeWait))
	if err != nil {
		return err
	}

	return wsc.ws.WriteMessage(mt, payload)
}

// here i would've probably choose one of the two options: strategies depending
// on msg type + handling some unknowns or routing + handlers based on action type
// well... still won't for test assignment
func (wsc *WsClient) handleMessage(msg map[string]interface{}) {
	action, ok := msg["action"].(string)

	if !ok {
		fmt.Println("No action in message")
		return
	}

	if action == "subscribe" {
		wsc.broadcst.AddConsumer(wsc.con)

		symbols, ok := msg["symbols"].([]interface{})

		if !ok {
			return
		}

		symbolsStr := make([]string, len(symbols))
		for i, symb := range symbols {
			symbolsStr[i] = symb.(string)
		}
		wsc.con.SetAcceptedSymbols(symbolsStr)
	}

	if action == "unsubscribe" {
		wsc.broadcst.RemoveConsumer(wsc.con)
	}
}