package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Counter represents a counter document in MongoDB
type Counter struct {
	ID    string `bson:"_id" json:"id"`
	Count int    `bson:"count" json:"count"`
}

// initializeCounters creates counter documents if they don't exist
func initializeCounters() {
	ctx := context.Background()
	countersCollection := db.Collection("counters")

	// Initialize webhook counter
	_, err := countersCollection.UpdateOne(
		ctx,
		bson.M{"_id": "webhook"},
		bson.M{"$setOnInsert": bson.M{"count": 0}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		log.Println("Error initializing webhook counter:", err)
	}

	// Initialize page view counter
	_, err = countersCollection.UpdateOne(
		ctx,
		bson.M{"_id": "pageviews"},
		bson.M{"$setOnInsert": bson.M{"count": 0}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		log.Println("Error initializing pageview counter:", err)
	}
}

// incrementHandler handles increment requests
func incrementHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()
	countersCollection := db.Collection("counters")

	_, err := countersCollection.UpdateOne(
		ctx,
		bson.M{"_id": "webhook"},
		bson.M{"$inc": bson.M{"count": 1}},
	)
	if err != nil {
		http.Error(w, "Error incrementing counter", http.StatusInternalServerError)
		return
	}

	// Get updated counter value
	var webhookCounter Counter
	err = countersCollection.FindOne(ctx, bson.M{"_id": "webhook"}).Decode(&webhookCounter)
	if err != nil {
		http.Error(w, "Error getting counter", http.StatusInternalServerError)
		return
	}

	// Broadcast to all WebSocket clients
	hub.broadcast <- CounterUpdate{Count: webhookCounter.Count}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CounterUpdate{Count: webhookCounter.Count})
}

// decrementHandler handles decrement requests
func decrementHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()
	countersCollection := db.Collection("counters")

	_, err := countersCollection.UpdateOne(
		ctx,
		bson.M{"_id": "webhook"},
		bson.M{"$inc": bson.M{"count": -1}},
	)
	if err != nil {
		http.Error(w, "Error decrementing counter", http.StatusInternalServerError)
		return
	}

	// Get updated counter value
	var webhookCounter Counter
	err = countersCollection.FindOne(ctx, bson.M{"_id": "webhook"}).Decode(&webhookCounter)
	if err != nil {
		http.Error(w, "Error getting counter", http.StatusInternalServerError)
		return
	}

	// Broadcast to all WebSocket clients
	hub.broadcast <- CounterUpdate{Count: webhookCounter.Count}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CounterUpdate{Count: webhookCounter.Count})
}
