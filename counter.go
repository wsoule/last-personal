package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CounterData represents data passed to counter partial template
type CounterData struct {
	Count       int
	TotalClicks int
}

// isHTMXRequest checks if the request came from HTMX
func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

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

	// Initialize total clicks counter
	_, err = countersCollection.UpdateOne(
		ctx,
		bson.M{"_id": "totalClicks"},
		bson.M{"$setOnInsert": bson.M{"count": 0}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		log.Println("Error initializing total clicks counter:", err)
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

	// Atomic increment and get updated value in one operation
	var webhookCounter Counter
	err := countersCollection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": "webhook"},
		bson.M{"$inc": bson.M{"count": 1}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&webhookCounter)
	if err != nil {
		http.Error(w, "Error incrementing counter", http.StatusInternalServerError)
		return
	}

	// Async increment total clicks counter (non-blocking)
	go func() {
		countersCollection.FindOneAndUpdate(
			context.Background(),
			bson.M{"_id": "totalClicks"},
			bson.M{"$inc": bson.M{"count": 1}},
			options.FindOneAndUpdate().SetReturnDocument(options.After),
		)
	}()

	// Get total clicks for broadcast
	var totalClicksCounter Counter
	err = countersCollection.FindOne(ctx, bson.M{"_id": "totalClicks"}).Decode(&totalClicksCounter)
	if err != nil {
		log.Println("Error getting total clicks:", err)
		totalClicksCounter.Count = 0
	}

	// Broadcast to all WebSocket clients
	update := CounterUpdate{
		Count:       webhookCounter.Count,
		TotalClicks: totalClicksCounter.Count,
	}
	hub.broadcast <- update

	// Return HTML fragment for HTMX, JSON for other clients
	if isHTMXRequest(r) {
		w.Header().Set("Content-Type", "text/html")
		data := CounterData{
			Count:       webhookCounter.Count,
			TotalClicks: totalClicksCounter.Count,
		}
		err := templates.ExecuteTemplate(w, "counter", data)
		if err != nil {
			log.Println("Error executing counter template:", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
		return
	}

	// Return JSON response for non-HTMX clients
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(update)
}

// decrementHandler handles decrement requests
func decrementHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()
	countersCollection := db.Collection("counters")

	// Atomic decrement and get updated value in one operation
	var webhookCounter Counter
	err := countersCollection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": "webhook"},
		bson.M{"$inc": bson.M{"count": -1}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&webhookCounter)
	if err != nil {
		http.Error(w, "Error decrementing counter", http.StatusInternalServerError)
		return
	}

	// Async increment total clicks counter (non-blocking)
	go func() {
		countersCollection.FindOneAndUpdate(
			context.Background(),
			bson.M{"_id": "totalClicks"},
			bson.M{"$inc": bson.M{"count": 1}},
			options.FindOneAndUpdate().SetReturnDocument(options.After),
		)
	}()

	// Get total clicks for broadcast
	var totalClicksCounter Counter
	err = countersCollection.FindOne(ctx, bson.M{"_id": "totalClicks"}).Decode(&totalClicksCounter)
	if err != nil {
		log.Println("Error getting total clicks:", err)
		totalClicksCounter.Count = 0
	}

	// Broadcast to all WebSocket clients
	update := CounterUpdate{
		Count:       webhookCounter.Count,
		TotalClicks: totalClicksCounter.Count,
	}
	hub.broadcast <- update

	// Return HTML fragment for HTMX, JSON for other clients
	if isHTMXRequest(r) {
		w.Header().Set("Content-Type", "text/html")
		data := CounterData{
			Count:       webhookCounter.Count,
			TotalClicks: totalClicksCounter.Count,
		}
		err := templates.ExecuteTemplate(w, "counter", data)
		if err != nil {
			log.Println("Error executing counter template:", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
		return
	}

	// Return JSON response for non-HTMX clients
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(update)
}
