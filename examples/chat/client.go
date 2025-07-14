package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Binit-Dhakal/ws"
)

// client is middleman between hub and websocket connection
type Client struct {
	hub  *Hub
	conn *ws.Conn
	send chan []byte
}

// readpump pumps message from websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Conn.Close()
	}()

	for {
		frame, err := c.conn.ReadFrame()
		if err != nil {
			if err == io.EOF {
				log.Println("Client disconnected gracefully")
			} else {
				log.Printf("Error while reading frame: %v", err)
			}
			break
		}
		switch frame.OpCode {
		case ws.OpcodeClose:
			return
		case ws.OpcodePong:
			fmt.Println("Received pong")
		case ws.OpcodeBinary:
		case ws.OpcodeText:
			message := bytes.TrimSpace(
				bytes.Replace(frame.Payload, []byte{'\n'}, []byte{' '}, -1),
			)
			c.hub.broadcast <- message
		default:
			fmt.Println("Unknown opcode", frame.OpCode)
			break
		}
	}
}

// writePump pumps messages from hub to the websocket connection
func (c *Client) writePump() {
	pingTicker := time.NewTicker(time.Second * 9)
	defer func() {
		pingTicker.Stop()
		c.conn.Conn.Close()
	}()

	pingChan := make(chan bool, 1)
	// Goroutine to send periodic message
	go func() {
		for range pingTicker.C {
			select {
			case pingChan <- true:
			default:
				// channel full skip this beat
			}
		}
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// the hub closed the channel
				c.conn.WriteCloseFrame()
				return
			}
			err := c.conn.WriteTextFrame(string(message))
			if err != nil {
				c.conn.WriteCloseFrame()
				return
			}
		case <-pingTicker.C:
			err := c.conn.WritePingFrame([]byte{})
			if err != nil {
				return
			}
		}
	}
}

func serveWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := ws.Upgrade(w, r)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}
