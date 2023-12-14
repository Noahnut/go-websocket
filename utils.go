package websocket

import (
	"crypto/sha1"
	"encoding/base64"
	"hash"
	"sync"
)

var uidKey = []byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11")

var shaPool = sync.Pool{
	New: func() interface{} {
		return sha1.New()
	},
}

func computeAcceptKey(challengeKeys []byte) []byte {
	h := shaPool.Get().(hash.Hash)
	h.Reset()
	h.Write(challengeKeys)
	h.Write(uidKey)
	return []byte(base64.StdEncoding.EncodeToString(h.Sum(nil)))
}

func isValidChallengeKeys(s []byte) bool {

	if len(s) == 0 {
		return false
	}

	decoded, err := base64.StdEncoding.DecodeString(string(s))

	return err == nil && len(decoded) == 16
}
