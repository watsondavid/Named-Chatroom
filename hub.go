package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Message struct {
	Sender  *Client
	Time    time.Time
	Content string
}

func (m Message) String() string {
	return fmt.Sprintf("%s @%s: %s", m.Sender.Name, m.Time.Format("Jan _2 15:04:05.000000"), m.Content)
}

type Hub struct {
	clients map[*Client]bool

	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Println("Registered a new client")
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Println("Unregistered a Client")
			}
		case message := <-h.broadcast:
			log.Println("Broadcasting message: ", message)
			stringifiedMessage := struct {
				Sender  string
				Time    string
				Content string
			}{
				message.Sender.Name,
				message.Time.Format(time.Kitchen),
				message.Content,
			}
			messageJson, err := json.Marshal(stringifiedMessage)
			if err != nil {
				log.Println("Failed to marshal message: ", message, ": ", err)
				continue
			}
			for client := range h.clients {
				select {
				case client.send <- messageJson:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
