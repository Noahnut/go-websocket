package websocket

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/valyala/fasthttp"
)

type MessageHandler func(c *Conn, data []byte)

type Server struct {
	messageHandler MessageHandler
}

func (s *Server) SetMessageHandler(messageHandler MessageHandler) {
	s.messageHandler = messageHandler
}

// Upgrade upgrade http connection to websocket connection
func (s *Server) Upgrade(ctx *fasthttp.RequestCtx) error {

	var err error

	// websocket header Connection value should be Upgrade
	if !ctx.Request.Header.ConnectionUpgrade() {
		err = errors.New("websocket header Connection value should be Upgrade")
		return err
	}

	// websocket METHOD must be GET
	if !bytes.Equal(ctx.Request.Header.Method(), GetString) {
		err = errors.New("websocket METHOD must be GET")
		return err
	}

	// websocket header Upgrade value should be websocket
	if !bytes.Equal(ctx.Request.Header.PeekBytes(upgradeString), webSocketString) {
		err = errors.New("websocket header Upgrade value should be websocket")
		return err
	}

	// websocket header Sec-WebSocket-Version value should be 13
	if !bytes.Equal(ctx.Request.Header.PeekBytes(websocketVersionString), websocketAcceptVersionString) {
		err = errors.New("websocket header Sec-WebSocket-Version value should be 13")
		ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
		return err
	}

	websocketKey := ctx.Request.Header.PeekBytes(websocketKeyString)

	// websocket header Sec-WebSocket-Key should be base64 and size is 16
	if !isValidChallengeKeys(websocketKey) {
		err = errors.New("websocket header Sec-WebSocket-Key should be base64 and size is 16 websocketKey" + string(websocketKey))
		ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
		return err
	}

	// compute Sec-WebSocket-Accept key
	acceptKey := computeAcceptKey(websocketKey)

	ctx.Response.Header.SetBytesKV(upgradeString, webSocketString)
	ctx.Response.Header.SetBytesKV(connectionString, upgradeString)
	ctx.Response.Header.SetBytesKV(websocketAcceptString, acceptKey)
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
		close(conn.WriteChan)
		for range conn.ReadChan {
		}

		for range conn.WriteChan {
		}

		conn.c.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("serverConn ctx done")
			return
		case frame := <-conn.ReadChan:
			fmt.Println("server read Frame", frame.String())
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
