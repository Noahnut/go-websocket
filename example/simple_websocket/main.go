package main

import (
	"fmt"
	"time"

	"github.com/Noahnut/websocket"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/pprofhandler"
)

// simple sample for websocket server and client
func websocketServer() {
	wsServer := websocket.Server{}

	messageHandler := func(c *websocket.Conn, isBinary bool, data []byte) {
		fmt.Println(string(data))
		c.Write([]byte("receive data from client"))
	}

	pingHandler := func(c *websocket.Conn, data []byte) {
		fmt.Println("receive ping from client")
		c.Pong()
	}

	wsServer.SetMessageHandler(messageHandler)
	wsServer.SetPingHandler(pingHandler)

	router := router.New()

	router.GET("/ws", wsServer.Upgrade)
	router.GET("/debug/pprof/{profile:*}", pprofhandler.PprofHandler)

	server := fasthttp.Server{
		Handler: router.Handler,
	}

	server.ListenAndServe(":8009")
}

func websocketClient() {
	client, err := websocket.NewClient("ws://localhost:8009/ws")

	if err != nil {
		panic(err)
	}

	client.Ping()

	frameType, payload, _ := client.Read()

	fmt.Println(frameType, string(payload))

	client.Write([]byte("hello world"))

	frameType, payload, _ = client.Read()

	fmt.Println(frameType, string(payload))

	frameType, status := client.Close()

	fmt.Printf("close frameType %d status %d\n", frameType, status)
}

func main() {

	go websocketServer()

	time.Sleep(1 * time.Second)

	go websocketClient()

	time.Sleep(5 * time.Second)
}
