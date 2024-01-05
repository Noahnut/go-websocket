package websocket

import (
	"bytes"
	"context"
	"net"

	"github.com/valyala/fasthttp"
)

type MessageHandler func(c *Conn, data []byte)

const (
	ErrorWebsocketHeaderConnectionValueShouldBeUpgrade = "websocket header Connection value should be Upgrade"
	ErrorWebsocketMethodMustBeGet                      = "websocket METHOD must be GET"
	ErrorWebsocketHeaderUpgradeValueShouldBeWebsocket  = "websocket header Upgrade value should be websocket"
	ErrorWebsocketHeaderSecWebSocketVersionValue       = "websocket header Sec-WebSocket-Version value should be 13"
	ErrorWebsocketHeaderSecWebSocketKey                = "websocket header Sec-WebSocket-Key should be base64 and size is 16"
	ErrorRequestOriginNotSameAsWebsocketOrigin         = "request origin not same as websocket origin"
)

// This is for debug goroutine leak
/*
func routineMonitor() {

		ticker := time.Tick(500 * time.Millisecond)

		for {
			select {
			case <-ticker:
				fmt.Fprintf(os.Stderr, "%d\n", runtime.NumGoroutine())
			}
		}
	}

	func init() {
		go routineMonitor()
	}
*/
type Server struct {
	CheckOrigin func(ctx *fasthttp.RequestCtx) bool

	messageHandler MessageHandler
}

func (s *Server) SetMessageHandler(messageHandler MessageHandler) {
	s.messageHandler = messageHandler
}

// Upgrade upgrade http connection to websocket connection
func (s *Server) Upgrade(ctx *fasthttp.RequestCtx) {
	// websocket header Connection value should be Upgrade
	if !ctx.Request.Header.ConnectionUpgrade() {
		ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.Response.SetBodyString(ErrorWebsocketHeaderConnectionValueShouldBeUpgrade)
		return
	}

	// websocket METHOD must be GET
	if !bytes.Equal(ctx.Request.Header.Method(), getString) {
		ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.Response.SetBodyString(ErrorWebsocketMethodMustBeGet)
		return
	}

	// websocket header Upgrade value should be websocket
	if !bytes.Equal(ctx.Request.Header.PeekBytes(upgradeString), webSocketString) {
		ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.Response.SetBodyString(ErrorWebsocketHeaderUpgradeValueShouldBeWebsocket)
		return
	}

	// websocket header Sec-WebSocket-Version value should be 13
	if !bytes.Equal(ctx.Request.Header.PeekBytes(websocketVersionString), websocketAcceptVersionString) {
		ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.Response.SetBodyString(ErrorWebsocketHeaderSecWebSocketVersionValue)
		return
	}

	if s.CheckOrigin == nil {
		s.CheckOrigin = checkSameOrigin
	}

	if !s.CheckOrigin(ctx) {
		ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.Response.SetBodyString(ErrorRequestOriginNotSameAsWebsocketOrigin)
		return
	}

	websocketKey := ctx.Request.Header.PeekBytes(websocketKeyString)

	// websocket header Sec-WebSocket-Key should be base64 and size is 16
	if !isValidChallengeKeys(websocketKey) {
		ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.Response.SetBodyString(ErrorWebsocketHeaderSecWebSocketKey)
		return
	}

	// compute Sec-WebSocket-Accept key
	acceptKey := computeAcceptKey(websocketKey)
	ctx.Response.Header.SetBytesKV(websocketAcceptString, acceptKey)

	ctx.Response.Header.SetBytesKV(upgradeString, webSocketString)
	ctx.Response.Header.SetBytesKV(connectionString, upgradeString)
	ctx.Response.SetStatusCode(fasthttp.StatusSwitchingProtocols)

	// hijack the connection to let's server handle the connection
	ctx.Hijack(func(c net.Conn) {

		ctx, cancel := context.WithCancel(context.Background())

		conn := NewConn(ctx, c, cancel)

		s.serverConn(ctx, conn)
	})

	return
}

func (s *Server) serverConn(ctx context.Context, conn *Conn) {

loop:
	for {
		select {
		case <-conn.ctx.Done():
			break loop
		case frame := <-conn.ReadChan:
			s.frameHandler(conn, frame)
			ReleaseFrame(frame)

			if conn.isClose {
				break loop
			}
		}
	}

	// clean all the channel data prevent goroutine leak
	conn.c.Close()

	for len(conn.ReadChan) > 0 {
		fr, ok := <-conn.ReadChan

		if !ok {
			break
		}

		if !fr.IsControl() {
			s.frameHandler(conn, fr)
		}

		ReleaseFrame(fr)
	}

	conn.waitGroup.Wait()
}

func (s *Server) frameHandler(conn *Conn, frame *Frame) {
	if frame.IsControl() {
		switch frame.frameType {
		case codeClose:
			s.closeHandler(conn, frame)
		case codePing:
			s.pingHandler(conn, frame)
		case codePong:
			s.pongHandler(conn, frame)
		}
	} else {
		s.dataFrameHandler(conn, frame)
	}
}

func (s *Server) closeHandler(conn *Conn, frame *Frame) {
	conn.Close()
}

func (s *Server) pingHandler(conn *Conn, frame *Frame) {
	conn.Pong()
}

func (s *Server) pongHandler(conn *Conn, frame *Frame) {

}

func (s *Server) dataFrameHandler(conn *Conn, frame *Frame) {
	s.messageHandler(conn, frame.payload)
}

func checkSameOrigin(ctx *fasthttp.RequestCtx) bool {

	origin := ctx.Request.Header.PeekBytes(originString)

	if len(origin) == 0 {
		return true
	}

	if bytes.Equal(origin, ctx.Host()) {
		return true
	}

	return false
}
