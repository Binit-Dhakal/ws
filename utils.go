package ws

import (
	"crypto/sha1"
	"encoding/base64"
)

var GUID = []byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11")

// Compute Accept Key from Sec-WS-Key+GUID and hashing and encoding it
func computeAcceptKey(key string) string {
	h := sha1.New()
	h.Write([]byte(key))
	h.Write(GUID)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func isValidSecKey(key string) bool {
	if key == "" {
		return false
	}
	s, err := base64.StdEncoding.DecodeString(key)
	return err == nil && len(s) == 16
}
