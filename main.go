package main

import (
	"log"
	"net/http"
)

type NewRule struct {
	NewRule string
}

func main() {
	// Serve React Front-end
	fs := http.FileServer(http.Dir("./my-app/build"))
	http.Handle("/", fs)

	// Serve websocket back end
	hub := newHub()
	go hub.run()
	http.HandleFunc("/api/connect", func(w http.ResponseWriter, r *http.Request) {
		hub.handleNewConnection(w, r)
	})

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
