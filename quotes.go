package main

import (
	"context"
	"net/http"
	"time"
)

// Quote represents a quote document in MongoDB
type Quote struct {
	Name      string    `bson:"name" json:"name"`
	Quote     string    `bson:"quote" json:"quote"`
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
}

// quoteHandler handles quote submission requests
func quoteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	quoteText := r.FormValue("quote")
	name := r.FormValue("name")

	if quoteText == "" {
		http.Error(w, "Quote cannot be empty", http.StatusBadRequest)
		return
	}

	if name == "" {
		name = "Unknown"
	}

	quote := Quote{
		Name:      name,
		Quote:     quoteText,
		Timestamp: time.Now(),
	}

	ctx := context.Background()
	quotesCollection := db.Collection("quotes")
	_, err = quotesCollection.InsertOne(ctx, quote)
	if err != nil {
		http.Error(w, "Error saving quote", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
