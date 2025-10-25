package main

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Hub maintains active WebSocket connections and broadcasts messages
type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan CounterUpdate
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
}

// CounterUpdate represents a counter value update
type CounterUpdate struct {
	Count       int `json:"count"`
	TotalClicks int `json:"totalClicks"`
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan CounterUpdate),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()
			log.Printf("WebSocket client connected. Total clients: %d", len(h.clients))

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected. Total clients: %d", len(h.clients))

		case update := <-h.broadcast:
			h.mu.Lock()
			for conn := range h.clients {
				err := conn.WriteJSON(update)
				if err != nil {
					log.Printf("WebSocket write error: %v", err)
					conn.Close()
					delete(h.clients, conn)
				}
			}
			h.mu.Unlock()
		}
	}
}

// wsHandler handles WebSocket connections
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Register the new client
	hub.register <- conn

	// Send current counter values to new client
	ctx := context.Background()
	countersCollection := db.Collection("counters")

	var webhookCounter Counter
	var totalClicksCounter Counter

	countersCollection.FindOne(ctx, bson.M{"_id": "webhook"}).Decode(&webhookCounter)
	countersCollection.FindOne(ctx, bson.M{"_id": "totalClicks"}).Decode(&totalClicksCounter)

	conn.WriteJSON(CounterUpdate{
		Count:       webhookCounter.Count,
		TotalClicks: totalClicksCounter.Count,
	})

	// Keep connection alive and handle cleanup
	defer func() {
		hub.unregister <- conn
	}()

	// Read messages (client shouldn't send any, but this keeps connection alive)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
