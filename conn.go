package ws

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Conn struct {
	Conn  net.Conn
	bufrw *bufio.ReadWriter
	mu    sync.Mutex
}

func (c *Conn) ReadFrame() (*Frame, error) {
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))

	c.bufrw.Writer.Flush()

	header := make([]byte, 2)
	_, err := io.ReadFull(c.bufrw, header)
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	frame := &Frame{}
	// first byte: FIN(1 byte) + RSV(3 byte) + OpCode(4 byte)
	frame.FIN = (header[0] & 0x80) != 0 // Most significant bits
	frame.OpCode = header[0] & 0x0F     // Lower 4 bits
	// second byte: Masked(1 byte) + payload Len(7 byte)
	frame.Masked = (header[1] & 0x80) != 0 // Lower 1 bits
	payloadLen := uint64(header[1] & 0x7F) // Lower 7 bits

	switch payloadLen {
	case 126: // next 2 bit define payload
		extLenBytes := make([]byte, 2)
		_, err := io.ReadFull(c.bufrw, extLenBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to read extended payload length (2 bytes): %w", err)
		}
		frame.PayloadLen = uint64(extLenBytes[0])<<8 | uint64(extLenBytes[1])
	case 127: // next 4 bytes define payload
		extLenBytes := make([]byte, 4)
		_, err := io.ReadFull(c.bufrw, extLenBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to read extended payload length (4 bytes): %w", err)
		}
		for i := 0; i < 8; i++ {
			frame.PayloadLen = (frame.PayloadLen << 8) | uint64(extLenBytes[i])
		}

	default:
		frame.PayloadLen = payloadLen
	}

	if frame.PayloadLen > maxMessageSize {
		return nil, fmt.Errorf("payload size exceeds limit")
	}

	// Read masking key (if masked)
	if frame.Masked {
		frame.MaskingKey = make([]byte, 4)
		_, err = io.ReadFull(c.bufrw, frame.MaskingKey)
		if err != nil {
			return nil, fmt.Errorf("failed to read payload: %w", err)
		}
	}

	// Read payload
	frame.Payload = make([]byte, frame.PayloadLen)
	_, err = io.ReadFull(c.bufrw, frame.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to read payload: %w", err)
	}

	// unmask payload if masked (client to server messages are always masked)
	if frame.Masked {
		for i := uint64(0); i < frame.PayloadLen; i++ {
			frame.Payload[i] ^= frame.MaskingKey[i%4]
		}
	}

	return frame, nil
}

func (c *Conn) writeFrame(opcode byte, payload []byte) error {
	c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
	c.mu.Lock()
	defer c.mu.Unlock()

	// Masking is applied only from client-to-server
	header := make([]byte, 2)
	// Byte 0: FIN(1 bit)=1 + RSV(3 bit)=0 + Opcode(4 bit)
	header[0] = 0x80 | opcode

	payloadLen := len(payload)
	switch {
	case payloadLen <= 125:
		header[1] = byte(payloadLen)
	case payloadLen <= 65535:
		header[1] = 126
		header = append(header, byte(payloadLen>>8), byte(payloadLen&0xFF))
	case payloadLen > 65535:
		header[1] = 127
		header = append(header,
			0, 0, 0, 0,
			byte(payloadLen>>56), byte(payloadLen>>48), byte(payloadLen>>40), byte(payloadLen>>32),
			byte(payloadLen>>24), byte(payloadLen>>16), byte(payloadLen>>8), byte(payloadLen&0xFF),
		)
	}

	_, err := c.bufrw.Write(header)
	if err != nil {
		return fmt.Errorf("failed to write frame header: %w", err)
	}

	// write payload
	_, err = c.bufrw.Write(payload)
	if err != nil {
		return fmt.Errorf("failed to write payload: %w", err)
	}

	return c.bufrw.Flush()
}

func (c *Conn) WriteTextFrame(text string) error {
	return c.writeFrame(OpcodeText, []byte(text))
}

func (c *Conn) WriteCloseFrame() error {
	return c.writeFrame(OpcodeClose, []byte{})
}

func (c *Conn) WritePingFrame(payload []byte) error {
	fmt.Println("Sending ping frame")
	return c.writeFrame(OpcodePing, payload)
}

func (c *Conn) WritePongFrame(payload []byte) error {
	return c.writeFrame(OpcodePong, payload)
}
