package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"server/internal/server"
	"server/internal/server/clients"
)

var (
	port = flag.String("port", "8080", "port to run the server on")
)

func main() {
	flag.Parse()
	hub := server.NewHub()

	// Start the hub first
	go hub.Run()
	log.Println("Hub is running and ready to accept connections")

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.Serve(clients.NewWebSocketClient, w, r)
	})

	// Start the server
	addr := fmt.Sprintf(":%s", *port)
	fmt.Printf("Server is listening on %s\n", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
