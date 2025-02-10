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
var broadcast = make(chan map[string]string)
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

type Message struct {
	Content string `json:"content"`
	Type    string `json:"type"`
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
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Println(red, "WebSocket read error:", err, reset)
			mutex.Lock()
			delete(clients, ws)
			mutex.Unlock()
			break
		}
		broadcast <- map[string]string{"content": msg.Content, "type": msg.Type}
	}
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	var msg Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Println(red, "Invalid POST request:", err, reset)
		return
	}

	log.Println(yellow, "Received POST request with message:", msg, reset)
	broadcast <- map[string]string{"content": msg.Content, "type": msg.Type}
	w.WriteHeader(http.StatusOK)
}

func handleBroadcast() {
	for {
		message := <-broadcast
		msgJSON, err := json.Marshal(message)
		if err != nil {
			log.Println(red, "Error marshalling message:", err, reset)
			continue
		}

		log.Println(green, "Broadcasting message:", string(msgJSON), reset)
		mutex.Lock()
		for client := range clients {
			if err := client.WriteMessage(websocket.TextMessage, msgJSON); err != nil {
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
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(red, "Server failed to start:", err, reset)
	}
}
