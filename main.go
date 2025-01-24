package main

import (
	"fmt"
	"log"
	"net/http"
	"github.com/gorilla/websocket"
	"encoding/json"
	"sync"
)

var clients = make(map[*websocket.Conn]bool) // 当前连接的客户端
var broadcast = make(chan string)             // 用于广播消息的通道
var mutex = &sync.Mutex{}                     // 确保线程安全

// 颜色控制
const (
	reset   = "\033[0m"
	green   = "\033[32m"
	red     = "\033[31m"
	yellow  = "\033[33m"
)

// 处理 WebSocket 连接
func handleConnections(w http.ResponseWriter, r *http.Request) {
	// 升级 HTTP 连接为 WebSocket 连接
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 允许跨域连接
		},
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(red, err, reset)
		return
	}
	defer ws.Close()

	// 将新连接加入 clients 集合
	mutex.Lock()
	clients[ws] = true
	mutex.Unlock()

	log.Println(green, "New WebSocket connection established", reset)

	// 一直监听该 WebSocket 连接上的消息
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Println(red, err, reset)
			mutex.Lock()
			delete(clients, ws)
			mutex.Unlock()
			break
		}
	}
}

// 处理 POST 请求，接收数据并推送给所有连接的客户端
func handlePost(w http.ResponseWriter, r *http.Request) {
	// 解析请求体
	var msg map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Println(red, "Invalid request received", reset)
		return
	}

	// 从请求中获取消息
	message, ok := msg["message"].(string)
	if !ok || message == "" {
		http.Error(w, "Message is required", http.StatusBadRequest)
		log.Println(red, "Message is required", reset)
		return
	}

	log.Println(yellow, "Received POST request with message:", message, reset)

	// 将消息广播给所有 WebSocket 客户端
	broadcast <- message
	w.WriteHeader(http.StatusOK)
}

// 广播所有接收到的消息
func handleBroadcast() {
	for {
		// 从 broadcast 通道接收消息
		message := <-broadcast

		log.Println(green, "Broadcasting message:", message, reset)

		// 向所有连接的客户端发送消息
		mutex.Lock()
		for client := range clients {
			if err := client.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
				log.Println(red, "Error sending message:", err, reset)
				client.Close()
				delete(clients, client)
			}
		}
		mutex.Unlock()
	}
}

func main() {
	// 设置 WebSocket 路由
	http.HandleFunc("/ws", handleConnections)

	// 设置 POST 请求路由
	http.HandleFunc("/post", handlePost)

	// 启动广播处理
	go handleBroadcast()

	// 启动 HTTP 服务器
	addr := "localhost:8080"
	fmt.Println("Server started on " + addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
