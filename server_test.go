package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestServer(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	// GIVEN a server is running
	hub := newHub()
	go hub.run()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.handleNewConnection(w, r)
	}))
	defer s.Close()

	// AND a pair of websocket clients are connected
	connect := func(name string) (*websocket.Conn, error) {
		u := "ws" + strings.TrimPrefix(s.URL, "http") + "?name=" + name
		ws, _, err := websocket.DefaultDialer.Dial(u, nil)
		if err != nil {
			return nil, err
		}
		return ws, nil
	}
	websockets := make([]*websocket.Conn, 2)
	for i := 0; i < 2; i++ {
		var err error
		websockets[i], err = connect("client" + fmt.Sprint(i))
		if err != nil {
			t.Fatalf("%v", err)
		}
	}

	// WHEN a message is sent over the websocket
	websockets[0].WriteMessage(websocket.TextMessage, []byte("test message"))

	// THEN the message is broadcast back to both clients
	for i := 0; i < 2; i++ {
		messageType, rawMessage, err := websockets[i].ReadMessage()
		if err != nil {
			t.Fatalf("%v", err)
		}
		if messageType != websocket.TextMessage {
			t.Fatal("Expeted Text Message")
		}
		var parsedMessage struct {
			Sender  string
			Time    string
			Content string
		}
		err = json.Unmarshal(rawMessage, &parsedMessage)
		if err != nil {
			t.Fatal("Failed to parse message: ", err)
		}
		if parsedMessage.Sender != "client0" {
			t.Fatalf("Sender name \"%s\" not as expected: %s", parsedMessage.Sender, "client0")
		}
		if parsedMessage.Time == "" {
			t.Fatalf("Message has empty timestamp")
		}
		if parsedMessage.Content != "test message" {
			t.Fatalf("Message content \"%s\" not as expected: %s", parsedMessage.Content, "test message")
		}
	}
}
