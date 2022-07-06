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

var newline = []byte{'\n'}

type WebsocketConnection struct {
	conn *websocket.Conn
}

func (ws *WebsocketConnection) ReadMessage() (int, []byte, error) {
	return ws.conn.ReadMessage()
}
func (ws *WebsocketConnection) Close() {
	ws.conn.Close()
}
func (ws *WebsocketConnection) WriteMessage(messageType int, message []byte) error {
	w, err := ws.conn.NextWriter(messageType)
	if err != nil {
		return err
	}
	w.Write(message)
	w.Write(newline)
	return w.Close()
}
func (ws *WebsocketConnection) SetWriteDeadline(d time.Duration) {
	ws.conn.SetWriteDeadline(time.Now().Add(d))
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

// Implements the interface defined by Client.go
func (h *Hub) Broadcast(m Message) {
	h.broadcast <- m
}
func (h *Hub) Register(c *Client) {
	h.register <- c
}
func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

func (h *Hub) handleNewConnection(w http.ResponseWriter, r *http.Request) {
	// Get name from query params.
	name := r.URL.Query().Get("name")
	log.Println(name, " has connected")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	client := &Client{hub: h, conn: &WebsocketConnection{conn}, Name: name, send: make(chan []byte, 256)}
	client.hub.Register(client)

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
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
