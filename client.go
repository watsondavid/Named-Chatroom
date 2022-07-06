package main

import (
	"bytes"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type IHub interface {
	Broadcast(Message)
	Register(*Client)
	Unregister(*Client)
}

type Connection interface {
	Close()
	ReadMessage() (int, []byte, error)
	WriteMessage(int, []byte) error
	SetWriteDeadline(time.Duration)
}

type Client struct {
	hub  IHub
	conn Connection
	Name string
	send chan []byte
}

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

var space = []byte{' '}

func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	for {
		_, text, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("error: ", err)
			}
			break
		}
		text = bytes.TrimSpace(bytes.Replace(text, newline, space, -1))
		message := Message{Sender: c, Time: time.Now(), Content: string(text)}
		c.hub.Broadcast(message)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(writeWait)
			if !ok {
				// the hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(writeWait)
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
