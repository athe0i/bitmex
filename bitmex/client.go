package bitmex

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/url"
)

// This struct serves as a client for the Bitmex,
// Constantly listens for the updates from bitmex and
// Pushes them to the channel.
// In prod grade app this should be a separate process/service writing to
// some data storage or at least queue so we don't loose any data when our web server dies.
type BitmexClient struct {
	endpoint string
	path string
	key string
	secret string
}

type Update struct {
	Symbol string
	LastPrice float64
	Timestamp string
}

func NewBitmexClient(endpoint string, path string, key string, secret string) *BitmexClient {
	return &BitmexClient{
		endpoint: endpoint,
		path:     path,
		key:      key,
		secret:   secret,
	}
}

// Opens ws connection to the Bitmex
// and starts process that pushes data to our data channel
func (bc *BitmexClient) Run(ctx context.Context, data chan Update) {
	u := url.URL{Scheme: "wss", Host: bc.endpoint, Path: bc.path, RawQuery: "subscribe=instrument"}
	fmt.Printf("connecting to %s\n", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Printf("dial:%s\n", err.Error())
	}
	defer c.Close()

	go bc.startReader(c, data)

	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}

// Reads messages from the socket connection and pushes desired ones into the
// data channel for processing
func (bc *BitmexClient) startReader(c *websocket.Conn, data chan <- Update) {
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			fmt.Printf("read err:%s\n", err)
			return
		}
		mp := make(map[string]interface{})

		err = json.Unmarshal(message, &mp)

		if err != nil {
			fmt.Printf("unmarshall err: %s\n", err.Error())
			continue
		}

		update := bc.processUpdate(mp)

		if update.LastPrice != 0 && update.Symbol != "" {
			data <- update
		}
	}
}

// Here we are filtering out only those updates, that we care about simply ignoring the rest
func (bc *BitmexClient) processUpdate(mp map[string]interface{}) (upd Update) {
	// here ideally we should create an action through some factory for any valid scenario, but since we don't really
	// care for anything but updates with lastPrice - lets filter out everything else and just use what we need
	if action, ok := mp["action"]; ok && action.(string) == "update" {
		data := mp["data"].([]interface{})[0]
		dataMap := data.(map[string]interface{})

		if dataMap["lastPrice"] != nil {
			upd.LastPrice = dataMap["lastPrice"].(float64)
			upd.Symbol = dataMap["symbol"].(string)
			upd.Timestamp = dataMap["timestamp"].(string)
		}
	}

	return
}