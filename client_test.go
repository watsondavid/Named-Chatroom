package main

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"gotest.tools/poll"
)

type Msg struct {
	messageType int
	message     []byte
}

type MockHub struct {
	broadcastedMessages []Message
}

func (h *MockHub) Broadcast(m Message) {
	h.broadcastedMessages = append(h.broadcastedMessages, m)
}
func (h *MockHub) Register(c *Client) {
	/// Not used for testing, Needed for interface.
}
func (h *MockHub) Unregister(c *Client) {
	/// Not used for testing, Needed for interface.
}

type MockConnection struct {
	sendToClient      chan Msg
	receiveFromClient chan Msg
	closed            bool
}

func makeMockConnection() MockConnection {
	return MockConnection{
		sendToClient:      make(chan Msg),
		receiveFromClient: make(chan Msg),
		closed:            false,
	}
}

func (c *MockConnection) Close() {
	c.closed = true
}
func (c *MockConnection) ReadMessage() (int, []byte, error) {
	if c.closed {
		return 0, nil, errors.New("Connection closed")
	}
	msg := <-c.sendToClient
	return msg.messageType, msg.message, nil
}
func (c *MockConnection) WriteMessage(messageType int, message []byte) error {
	if c.closed {
		return errors.New("Connection closed")
	}
	c.receiveFromClient <- Msg{messageType, message}
	return nil
}
func (c *MockConnection) SetWriteDeadline(time.Duration) {
	/// Not used for testing, Needed for interface.
}

func TestReadPump(t *testing.T) {
	// GIVEN a client
	mockHub := MockHub{}
	mockConn := makeMockConnection()

	client := Client{hub: &mockHub, conn: &mockConn, Name: "test client", send: make(chan []byte)}

	// AND its ReadPump method is running
	go client.readPump()

	// WHEN a message is received from the websocket
	msg := Msg{websocket.TextMessage, []byte("test message")}
	mockConn.sendToClient <- msg

	check := func(t poll.LogT) poll.Result {
		if len(mockHub.broadcastedMessages) > 0 {
			return poll.Success()
		}
		return poll.Continue("waiting...")
	}
	poll.WaitOn(t, check, poll.WithTimeout(time.Second*10), poll.WithDelay(time.Millisecond*50))

	// THEN the message is Broadcast to the hub
	if len(mockHub.broadcastedMessages) != 1 {
		t.Fatal("Expected a message to be broadcast to the hub: ", len(mockHub.broadcastedMessages))
	}
	if mockHub.broadcastedMessages[0].Content != string(msg.message) {
		t.Fatalf("%+v did not match expected: %+v", mockHub.broadcastedMessages[0], msg)
	}
}

func TestWritePump(t *testing.T) {
	// GIVEN a client
	mockHub := MockHub{}
	mockConn := makeMockConnection()

	client := Client{hub: &mockHub, conn: &mockConn, Name: "test client", send: make(chan []byte)}

	// AND its WritePump method is running
	go client.writePump()

	// When a message is sent
	testMessage := []byte("test message")
	client.send <- testMessage

	// THEN it is written over the websocket
	var msg Msg
	check := func(t poll.LogT) poll.Result {
		select {
		case msg = <-mockConn.receiveFromClient:
			return poll.Success()
		default:
			return poll.Continue("waiting...")
		}
	}
	poll.WaitOn(t, check, poll.WithTimeout(time.Second*10), poll.WithDelay(time.Millisecond*50))

	if msg.messageType != websocket.TextMessage {
		t.Fatal("Expected a text message")
	}
	if !reflect.DeepEqual(msg.message, testMessage) {
		t.Fatalf("%s did not match expected message: %s", msg.message, testMessage)
	}
}
