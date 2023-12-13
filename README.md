
# Websocket

websocket implement by [fasthttp](https://github.com/valyala/fasthttp)


# Example

## Server Side

```go
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



```

## Client

```go
func websocketClient() {
	client, err := websocket.NewClient("ws://localhost:8009")

	if err != nil {
		panic(err)
	}

	client.Write([]byte("hello world"))

	fmt.Println(string(client.Read()))

	client.Close()
}
```