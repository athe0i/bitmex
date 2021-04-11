package main

import (
	"bitmex/bitmex"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
)

var bitmexClient *bitmex.BitmexClient
var broadcaster *Broadcaster

func init() {
	bitmexClient = bitmex.NewBitmexClient("testnet.bitmex.com", "/realtime", "not_used", "not_used")
	broadcaster = NewBroadcaster()
}

// Tbh all this code is far from ideal.
// Basically it just simply does its job. No SOLID, no advanced principles or patterns.
// I am not really sure if its any good.
func main() {
	dataChn := make(chan bitmex.Update)
	clientContext, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runBitmexClient(clientContext, dataChn)
	go runBroadcaster(clientContext, dataChn)

	router := gin.New()

	// following stuff should be really separate if we have any type of a bigger app.
	// and probably subdivided into some modules.... but hey, two endpoints... not
	// so much to talk about
	router.LoadHTMLFiles("index.html")

	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	router.GET("/ws", func(c *gin.Context) {
		wsc, err := NewWsClient(c.Writer, c.Request, broadcaster)
		if err != nil {
			fmt.Println(err)
			return
		}

		wsc.Serve(clientContext)
	})

	router.Run("0.0.0.0:8080")
}

func runBitmexClient(ctx context.Context, data chan bitmex.Update) {
	bitmexClient.Run(ctx, data)

	select {
	case <-ctx.Done():
		return
	}
}

func runBroadcaster(ctx context.Context, data chan bitmex.Update) {
	broadcaster.ListenForUpdates(ctx, data)

	select {
	case <-ctx.Done():
		return
	}
}