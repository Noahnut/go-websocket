package main

import (
	"fmt"
	"time"

	"github.com/Noahnut/websocket"
	"github.com/valyala/fasthttp"
)

// simple sample for websocket server and client
func websocketServer() {
	wsServer := websocket.Server{}

	messageHandler := func(c *websocket.Conn, data []byte) {
		fmt.Println(string(data))

		c.Write([]byte("receive data from client"))
	}

	wsServer.SetMessageHandler(messageHandler)

	fasthttp.ListenAndServe(":8009", func(ctx *fasthttp.RequestCtx) {
		err := wsServer.Upgrade(ctx)

		if err != nil {
			panic(err)
		}
	})
}

func websocketClient() {
	client, err := websocket.NewClient("ws://localhost:8009")

	if err != nil {
		panic(err)
	}

	client.Write([]byte("hello world"))

	fmt.Println(string(client.Read()))

	client.Close()
}

func main() {

	go websocketServer()

	time.Sleep(1 * time.Second)

	go websocketClient()

	time.Sleep(5 * time.Second)
}
