package websocket

import (
	"bytes"
	"context"
	"errors"
	"net"

	"github.com/valyala/fasthttp"
)

type MessageHandler func(c *Conn, data []byte)

var (
	ErrorWebsocketHeaderConnectionValueShouldBeUpgrade error = errors.New("websocket header Connection value should be Upgrade")
	ErrorWebsocketMethodMustBeGet                            = errors.New("websocket METHOD must be GET")
	ErrorWebsocketHeaderUpgradeValueShouldBeWebsocket        = errors.New("websocket header Upgrade value should be websocket")
	ErrorWebsocketHeaderSecWebSocketVersionValue             = errors.New("websocket header Sec-WebSocket-Version value should be 13")
	ErrorWebsocketHeaderSecWebSocketKey                      = errors.New("websocket header Sec-WebSocket-Key should be base64 and size is 16")
	ErrorRequestOriginNotSameAsWebsocketOrigin               = errors.New("request origin not same as websocket origin")
)

type Server struct {
	CheckOrigin func(ctx *fasthttp.RequestCtx) bool

	messageHandler MessageHandler
}

func (s *Server) SetMessageHandler(messageHandler MessageHandler) {
	s.messageHandler = messageHandler
}

// Upgrade upgrade http connection to websocket connection
func (s *Server) Upgrade(ctx *fasthttp.RequestCtx) error {
	// websocket header Connection value should be Upgrade
	if !ctx.Request.Header.ConnectionUpgrade() {
		return ErrorWebsocketHeaderConnectionValueShouldBeUpgrade
	}

	// websocket METHOD must be GET
	if !bytes.Equal(ctx.Request.Header.Method(), getString) {
		return ErrorWebsocketMethodMustBeGet
	}

	// websocket header Upgrade value should be websocket
	if !bytes.Equal(ctx.Request.Header.PeekBytes(upgradeString), webSocketString) {
		return ErrorWebsocketHeaderUpgradeValueShouldBeWebsocket
	}

	// websocket header Sec-WebSocket-Version value should be 13
	if !bytes.Equal(ctx.Request.Header.PeekBytes(websocketVersionString), websocketAcceptVersionString) {
		return ErrorWebsocketHeaderSecWebSocketVersionValue
	}

	if s.CheckOrigin == nil {
		s.CheckOrigin = checkSameOrigin
	}

	if !s.CheckOrigin(ctx) {
		return ErrorRequestOriginNotSameAsWebsocketOrigin
	}

	websocketKey := ctx.Request.Header.PeekBytes(websocketKeyString)

	// websocket header Sec-WebSocket-Key should be base64 and size is 16
	if !isValidChallengeKeys(websocketKey) {
		return ErrorWebsocketHeaderSecWebSocketKey
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

	return nil
}

func (s *Server) serverConn(ctx context.Context, conn *Conn) {

	defer func() {
		conn.waitGroup.Wait()

		// clean all the channel data prevent goroutine leak
		close(conn.ReadChan)

		// clean all the frame data
		for frame := range conn.ReadChan {
			s.frameHandler(conn, frame)
		}

		conn.c.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case frame := <-conn.ReadChan:
			s.frameHandler(conn, frame)
		}
	}

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
	conn.cancel()
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
