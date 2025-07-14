package main

import (
	"fmt"
	"sync"
)

type Hub struct {
	clients    map[*Client]bool
	mu         sync.Mutex
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			fmt.Println("Registering client", client)
			h.clients[client] = true
		case client := <-h.unregister:
			fmt.Println("unregistering the client", client)
			close(client.send)
			delete(h.clients, client)
		case message := <-h.broadcast:
			fmt.Println("Broadcasting message to", len(h.clients), "clients")
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// if unable to send
					fmt.Println("Removing client due to full send buffer", client)
					delete(h.clients, client)
				}
			}
		}
	}
}
