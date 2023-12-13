package websocket

import (
	"fmt"
	"sync"
	"testing"

	"github.com/valyala/fasthttp"
)

func Test_SimpleWsServer(t *testing.T) {
	wsServer := Server{}

	waitGroup := sync.WaitGroup{}

	waitGroup.Add(2)

	go func() {
		defer waitGroup.Done()

		fasthttp.ListenAndServe(":8009", func(ctx *fasthttp.RequestCtx) {
			err := wsServer.Upgrade(ctx)

			if err != nil {
				fmt.Println(err)
			}

		})
	}()

	waitGroup.Wait()

}
