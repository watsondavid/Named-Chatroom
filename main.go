package main

import (
	"log"
	"net/http"
)

type NewRule struct {
	NewRule string
}

// connects new websocket connections from the peer.
func handleWsConnection(hub *Hub, w http.ResponseWriter, r *http.Request) {
	// Get name from query params.
	name := r.URL.Query().Get("name")
	log.Println(name, " has connected")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: hub, conn: conn, Name: name, send: make(chan []byte, 256)}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}

func main() {
	// Serve React Front-end
	fs := http.FileServer(http.Dir("./my-app/build"))
	http.Handle("/", fs)

	// Serve websocket back end
	hub := newHub()
	go hub.run()
	http.HandleFunc("/api/connect", func(w http.ResponseWriter, r *http.Request) {
		handleWsConnection(hub, w, r)
	})

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
