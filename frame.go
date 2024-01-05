package websocket

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

var maskZeroBytes = []byte{0, 0, 0, 0}

var payloadZeroBytes = func() []byte {
	b := make([]byte, 128)
	for i := range b {
		b[i] = 0
	}
	return b
}()

const (
	finBit = byte(1 << 7)
	rsv1   = byte(1 << 6)
	rsv2   = byte(1 << 5)
	rsv3   = byte(1 << 4)
	mask   = byte(1 << 7)
)

type Frame struct {
	isFin       bool
	rsv1        bool
	rsv2        bool
	rsv3        bool
	mask        bool
	frameType   frameTypeCode
	payloadSize int64
	maskKey     []byte
	payload     []byte
}

var framePool = sync.Pool{
	New: func() interface{} {
		return newFrame()
	},
}

func newFrame() *Frame {
	return &Frame{
		maskKey: make([]byte, 4),
		payload: make([]byte, 0, 128),
	}
}

func AcquireFrame() *Frame {
	return framePool.Get().(*Frame)
}

func ReleaseFrame(f *Frame) {
	f.Reset()
	framePool.Put(f)
}

func (f *Frame) Reset() {
	f.isFin = false
	f.rsv1 = false
	f.rsv2 = false
	f.rsv3 = false
	f.mask = false
	f.frameType = codeUnknown
	f.payloadSize = 0
	copy(f.maskKey, maskZeroBytes)
	copy(f.payload, payloadZeroBytes)
}

func (f *Frame) String() string {
	return fmt.Sprintf("isFin: %v, rsv1: %v, rsv2: %v, rsv3: %v, frameType: %v, payloadSize: %v, payload: %v", f.isFin, f.rsv1, f.rsv2, f.rsv3, f.frameType.String(), f.payloadSize, string(f.payload))
}

func (f *Frame) IsFin() bool {
	return f.isFin
}

func (f *Frame) GetFrameType() frameTypeCode {
	return f.frameType
}

func (f *Frame) GetPayload() []byte {
	return f.payload
}

func (f *Frame) IsPing() bool {
	return f.frameType == codePing
}

func (f *Frame) IsPong() bool {
	return f.frameType == codePong
}

func (f *Frame) IsClose() bool {
	return f.frameType == codeClose
}

func (f *Frame) IsContinuation() bool {
	return f.frameType == codeContinuation
}

func (f *Frame) IsControl() bool {
	return f.IsPing() || f.IsPong() || f.IsClose()
}

func (f *Frame) SetFrameType(frameType frameTypeCode) {
	f.frameType = frameType
}

func (f *Frame) SetPayload(payload []byte) {
	f.payload = payload
}

func (f *Frame) SetPayloadSize(payloadSize int64) {
	f.payloadSize = payloadSize
}

func (f *Frame) SetFin() {
	f.isFin = true
}

func (f *Frame) SetStatus(status websocketStatusCode) {
	f.payload = make([]byte, 2)
	binary.BigEndian.PutUint16(f.payload, uint16(status))
	f.payloadSize = 2
}

func (f *Frame) WriteTo(wr io.Writer) (int64, error) {

	var n int64

	header := make([]byte, 2)

	if f.isFin {
		header[0] |= finBit
	}

	if f.rsv1 {
		header[0] |= rsv1
	}

	if f.rsv2 {
		header[0] |= rsv2
	}

	if f.rsv3 {
		header[0] |= rsv3
	}

	header[0] |= byte(f.frameType)

	if f.mask {
		header[1] |= mask
		// TODO
		// if mask should set the mask key
	}

	var payloadLenBytes []byte
	switch {
	case f.payloadSize > 65535:
		header[1] |= 127
		payloadLenBytes = make([]byte, 8)
		binary.BigEndian.PutUint64(payloadLenBytes, uint64(f.payloadSize))
	case f.payloadSize > 125:
		header[1] |= 126
		payloadLenBytes = make([]byte, 2)
		binary.BigEndian.PutUint16(payloadLenBytes, uint16(f.payloadSize))
	default:
		header[1] |= byte(f.payloadSize)
	}

	header = append(header, payloadLenBytes...)

	ni, err := wr.Write(header)

	if err != nil {
		return int64(ni), err
	}

	n += int64(ni)

	if f.mask {

		ni, err := wr.Write(f.maskKey)

		if err != nil {
			return 0, err
		}

		n += int64(ni)
	}

	if len(f.payload) > 0 {
		ni, err = wr.Write(f.payload)

		if err != nil {
			return int64(ni), err
		}

		n += int64(ni)
	}

	return n, err
}

func (f *Frame) UnMask() {
	for i := range f.payload {
		f.payload[i] ^= f.maskKey[i&3]
	}
}

func (f *Frame) ReadFrom(r io.Reader) (int64, error) {
	var err error
	var n, _ int

	header := make([]byte, 2)

	n, err = io.ReadFull(r, header)

	if err == io.ErrUnexpectedEOF {
		return int64(n), err
	}

	f.isFin = header[0]&finBit == finBit
	f.rsv1 = header[0]&rsv1 == rsv1
	f.rsv2 = header[0]&rsv2 == rsv2
	f.rsv3 = header[0]&rsv3 == rsv3
	f.frameType = frameTypeCode(header[0] & 0x0F)
	f.mask = header[1]&mask == mask
	f.payloadSize = int64(header[1] & 127)

	switch f.payloadSize {
	case 126:
		payloadSizeBytes := make([]byte, 2)
		n, err = io.ReadFull(r, payloadSizeBytes)

		if err != nil {
			return int64(n), err
		}

		f.payloadSize = int64(binary.BigEndian.Uint16(payloadSizeBytes))

	case 127:
		payloadSizeBytes := make([]byte, 8)
		n, err = io.ReadFull(r, payloadSizeBytes)

		if err != nil {
			return int64(n), err
		}

		f.payloadSize = int64(binary.BigEndian.Uint64(payloadSizeBytes))
	}

	if f.mask {
		n, err = io.ReadFull(r, f.maskKey)

		if err != nil {
			return int64(n), err
		}
	}

	f.payload = make([]byte, f.payloadSize)

	n, err = io.ReadFull(r, f.payload)

	if err != nil {
		return int64(n), err
	}

	if f.mask {
		f.UnMask()
	}

	return int64(n), err
}
