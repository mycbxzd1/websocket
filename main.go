package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan string)
var mutex = &sync.Mutex{}

const (
	reset  = "\033[0m"
	green  = "\033[32m"
	red    = "\033[31m"
	yellow = "\033[33m"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(red, "WebSocket upgrade failed:", err, reset)
		return
	}
	defer ws.Close()

	mutex.Lock()
	clients[ws] = true
	mutex.Unlock()

	log.Println(green, "New WebSocket connection established", reset)

	go handlePing(ws)

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Println(red, "WebSocket read error:", err, reset)
			mutex.Lock()
			delete(clients, ws)
			mutex.Unlock()
			break
		}
	}
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	var msg map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Println(red, "Invalid POST request:", err, reset)
		return
	}

	message, ok := msg["message"].(string)
	if !ok || message == "" {
		http.Error(w, "Message is required", http.StatusBadRequest)
		log.Println(red, "Message is required in POST", reset)
		return
	}

	log.Println(yellow, "Received POST request with message:", message, reset)
	broadcast <- message
	w.WriteHeader(http.StatusOK)
}

func handleBroadcast() {
	for {
		message := <-broadcast

		log.Println(green, "Broadcasting message:", message, reset)

		mutex.Lock()
		for client := range clients {
			if err := client.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
				log.Println(red, "Error broadcasting message:", err, reset)
				client.Close()
				delete(clients, client)
			}
		}
		mutex.Unlock()
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintln(w, "<h1>WebSocket server is running</h1>")
}

func handlePing(ws *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println(red, "Ping failed:", err, reset)
				return
			}
		}
	}
}

func main() {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/post", handlePost)

	go handleBroadcast()

	addr := ":15542"
	log.Println(green, "Server started on", addr, reset)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(red, "Server failed to start:", err, reset)
	}
}
