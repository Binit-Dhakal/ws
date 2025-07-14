package ws

import "time"

const (
	OpcodeContinous = 0x0
	OpcodeText      = 0x1
	OpcodeBinary    = 0x2
	OpcodeClose     = 0x8
	OpcodePing      = 0x9
	OpcodePong      = 0xA

	writeWait  = 10 * time.Second    // Time allowed to write a message to the peer.
	pongWait   = 60 * time.Second    // Time allowed to read the next pong message from the peer.
	pingPeriod = (pongWait * 9) / 10 // Send pings to peer with this period. Must be less than pongWait.

	maxMessageSize = 10 * 1024 * 1024 // Maximum message size allowed from peer. (10MB)
)
