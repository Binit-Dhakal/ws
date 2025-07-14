package main

import (
	"bytes"
	"github.com/Binit-Dhakal/ws"
	"log"
	"net/http"
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
			log.Printf("Error while reading frame: %v", err)
			break
		}
		message := bytes.TrimSpace(
			bytes.Replace(frame.Payload, []byte{'\n'}, []byte{' '}, -1),
		)
		c.hub.broadcast <- message
	}
}

// writePump pumps messages from hub to the websocket connection
func (c *Client) writePump() {
	defer func() {
		c.conn.Conn.Close()
	}()

	for {
		message, ok := <-c.send
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
