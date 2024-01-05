package websocket

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"unsafe"

	"github.com/valyala/fasthttp"
)

type Client struct {
	c        net.Conn
	rwBuffer *bufio.ReadWriter
}

func NewClient(url string) (*Client, error) {

	uri := fasthttp.AcquireURI()
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseURI(uri)
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	uri.Update(url)

	uri.SetScheme("http")

	addr := make([]byte, 0)

	addr = append(addr[:0], uri.Host()...)
	if n := bytes.LastIndexByte(addr, ':'); n == -1 {
		addr = append(addr, []byte(":80")...)
	}

	c, err := net.Dial("tcp", *(*string)(unsafe.Pointer(&addr)))

	if err != nil {
		panic(err)
	}

	req.Header.SetMethod("GET")
	req.Header.AddBytesKV(connectionString, upgradeString)
	req.Header.AddBytesKV(upgradeString, webSocketString)
	req.Header.AddBytesKV(websocketVersionString, websocketAcceptVersionString)
	req.Header.AddBytesKV(websocketKeyString, secWebsocketKeyValueString)

	req.SetRequestURIBytes(uri.FullURI())

	br := bufio.NewReader(c)
	bw := bufio.NewWriter(c)
	req.Write(bw)
	bw.Flush()

	err = resp.Read(br)
	if err == nil {
		if resp.StatusCode() == 101 {
			fmt.Println("upgrade success")
		}
	}

	websocketConn := &Client{
		c:        c,
		rwBuffer: bufio.NewReadWriter(br, bw),
	}

	return websocketConn, nil
}

func (c *Client) Write(p []byte) {

	frame := AcquireFrame()
	defer ReleaseFrame(frame)

	frame.SetFin()
	frame.SetFrameType(codeText)
	frame.SetPayload(p)
	frame.SetPayloadSize(int64(len(p)))

	if _, err := frame.WriteTo(c.rwBuffer); err == nil {
		c.rwBuffer.Flush()
	}
}

func (c *Client) Read() (frameTypeCode, []byte, error) {
	newFrame := newFrame()

	if _, err := newFrame.ReadFrom(c.rwBuffer); err != nil {
		return codeUnknown, nil, err
	}

	return newFrame.GetFrameType(), newFrame.GetPayload(), nil
}

func (c *Client) Close() (frameTypeCode, websocketStatusCode) {
	frame := AcquireFrame()
	defer ReleaseFrame(frame)

	frame.SetFin()
	frame.SetFrameType(codeClose)
	frame.SetStatus(websocketStatusCodeNormalClosure)

	if _, err := frame.WriteTo(c.rwBuffer); err == nil {
		c.rwBuffer.Flush()
	}

	frameType, payload, _ := c.Read()

	c.c.Close()

	status := websocketStatusCode(binary.BigEndian.Uint16(payload))

	return frameType, status
}

func (c *Client) Ping() {
	frame := AcquireFrame()
	defer ReleaseFrame(frame)

	frame.SetFin()
	frame.SetFrameType(codePing)

	if _, err := frame.WriteTo(c.rwBuffer); err == nil {
		c.rwBuffer.Flush()
	}
}
