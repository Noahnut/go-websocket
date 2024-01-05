package websocket

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"
)

const (
	readChanSize  = 128
	writeChanSize = 128
)

type Conn struct {
	c net.Conn

	isClose bool

	waitGroup sync.WaitGroup

	bufferReader *bufio.Reader
	bufferWriter *bufio.Writer

	ctx    context.Context
	cancel context.CancelFunc

	ReadChan  chan *Frame
	WriteChan chan *Frame

	wg sync.WaitGroup
}

func NewConn(ctx context.Context, conn net.Conn, cancel context.CancelFunc) *Conn {

	c := &Conn{
		c:            conn,
		ctx:          ctx,
		cancel:       cancel,
		bufferReader: bufio.NewReader(conn),
		bufferWriter: bufio.NewWriter(conn),
		ReadChan:     make(chan *Frame, readChanSize),
		WriteChan:    make(chan *Frame, writeChanSize),
	}

	c.waitGroup.Add(2)
	go c.readLoop()
	go c.writeLoop()

	return c
}

func (c *Conn) readLoop() {
	for {

		newFrame := AcquireFrame()

		_, err := newFrame.ReadFrom(c.bufferReader)

		if err != nil {
			c.isClose = true
			c.cancel()

			ReleaseFrame(newFrame)
			break
		}

		c.ReadChan <- newFrame

		// receive close frame just end readLoop routine
		if newFrame.IsClose() || c.isClose {
			c.isClose = true
			break
		}

	}

	c.waitGroup.Done()
}

func (c *Conn) writeLoop() {
loop:
	for {
		select {
		case frame := <-c.WriteChan:
			if _, err := frame.WriteTo(c.bufferWriter); err == nil {
				c.bufferWriter.Flush()
			} else {
				c.isClose = true
				c.cancel()
				break loop
			}

			ReleaseFrame(frame)

			if frame.IsClose() {
				break loop
			}
		case <-c.ctx.Done():
			break loop
		}
	}

	for len(c.WriteChan) > 0 {
		fr, ok := <-c.WriteChan

		if !ok {
			break
		}

		if _, err := fr.WriteTo(c.bufferWriter); err != nil {
			break
		}

		ReleaseFrame(fr)
	}

	c.waitGroup.Done()
}

func (c *Conn) Write(p []byte) (int, error) {

	frame := AcquireFrame()

	frame.SetFrameType(codeText)
	frame.SetPayload(p)
	frame.SetFin()
	frame.SetPayloadSize(int64(len(p)))

	if c.isClose {
		return 0, fmt.Errorf("conn is closed")
	}

	c.WriteChan <- frame

	return len(p), nil
}

func (c *Conn) Close() {

	frame := AcquireFrame()

	frame.SetStatus(websocketStatusCodeNormalClosure)
	frame.SetFrameType(codeClose)
	frame.SetFin()

	c.WriteChan <- frame

}

func (c *Conn) Ping() {
	frame := AcquireFrame()

	frame.SetFrameType(codePing)
	frame.SetFin()

	c.WriteChan <- frame
}

func (c *Conn) Pong() {
	frame := AcquireFrame()

	frame.SetFrameType(codePong)
	frame.SetFin()

	c.WriteChan <- frame
}

func (c *Conn) writeFrame(frame *Frame) {
	c.WriteChan <- frame
}
