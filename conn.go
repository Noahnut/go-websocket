package websocket

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"
)

const (
	readChanSize  = 1024
	writeChanSize = 1024
)

type Conn struct {
	c net.Conn

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
	defer c.waitGroup.Done()

	for {

		newFrame := NewFrame()

		n, err := newFrame.ReadFrom(c.bufferReader)

		if err != nil {
			fmt.Println(n, err)
		}

		c.ReadChan <- newFrame

		// receive close frame just end readLoop routine
		if newFrame.IsClose() {
			fmt.Println("readLoop close")
			return
		}

	}
}

func (c *Conn) writeLoop() {
	defer c.waitGroup.Done()

	for {
		select {
		case frame := <-c.WriteChan:
			if _, err := frame.WriteTo(c.bufferWriter); err == nil {
				c.bufferWriter.Flush()
			}

		case <-c.ctx.Done():
			fmt.Println("writeLoop ctx done")
			return
		default:
		}
	}
}

func (c *Conn) Write(p []byte) (int, error) {

	frame := NewFrame()

	frame.SetFrameType(codeText)
	frame.SetPayload(p)
	frame.SetFin()
	frame.SetPayloadSize(int64(len(p)))

	c.WriteChan <- frame

	return len(p), nil
}

func (c *Conn) Close() {

	frame := NewFrame()

	frame.SetStatus(websocketStatusCodeNormalClosure)
	frame.SetFrameType(codeClose)
	frame.SetFin()

	c.WriteChan <- frame

}

func (c *Conn) Ping() {
	frame := NewFrame()

	frame.SetFrameType(codePing)
	frame.SetFin()

	c.WriteChan <- frame
}

func (c *Conn) Pong() {
	frame := NewFrame()

	frame.SetFrameType(codePong)
	frame.SetFin()

	c.WriteChan <- frame
}

func (c *Conn) writeFrame(frame *Frame) {
	c.WriteChan <- frame
}
