package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client    *mongo.Client
	db        *mongo.Database
	templates *template.Template
	hub       *Hub
)

// GitHubRepo represents a GitHub repository
type GitHubRepo struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	HTMLURL         string `json:"html_url"`
	Language        string `json:"language"`
	StargazersCount int    `json:"stargazers_count"`
}

// PageData represents the data passed to the home page template
type PageData struct {
	Name          string
	WebhookCount  int
	PageViewCount int
	TotalClicks   int
	Quotes        []Quote
	GitHubRepos   []GitHubRepo
}

func main() {
	// Get MongoDB URI from environment
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	// Connect to MongoDB with connection pooling for concurrency
	var err error
	clientOptions := options.Client().
		ApplyURI(mongoURI).
		SetMaxPoolSize(100).    // Max 100 concurrent connections
		SetMinPoolSize(10)       // Keep 10 warm connections

	client, err = mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	// Test connection
	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.Fatal("Could not connect to MongoDB:", err)
	}

	db = client.Database("personal_website")

	// Initialize counters if they don't exist
	initializeCounters()

	// Parse templates
	templates = template.Must(template.ParseGlob("templates/*.html"))

	// Initialize and start WebSocket hub
	hub = NewHub()
	go hub.Run()

	// Routes
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/increment", incrementHandler)
	http.HandleFunc("/decrement", decrementHandler)
	http.HandleFunc("/quote", rateLimitMiddleware(quoteHandler, 5)) // 5 requests per minute
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/robots.txt")
	})
	http.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/sitemap.xml")
	})
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// homeHandler renders the home page
func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	ctx := context.Background()

	// Increment page view counter
	countersCollection := db.Collection("counters")
	_, err := countersCollection.UpdateOne(
		ctx,
		bson.M{"_id": "pageviews"},
		bson.M{"$inc": bson.M{"count": 1}},
	)
	if err != nil {
		log.Println("Error incrementing page views:", err)
	}

	// Get webhook counter
	var webhookCounter Counter
	err = countersCollection.FindOne(ctx, bson.M{"_id": "webhook"}).Decode(&webhookCounter)
	if err != nil {
		log.Println("Error getting webhook counter:", err)
		webhookCounter.Count = 0
	}

	// Get page view counter
	var pageViewCounter Counter
	err = countersCollection.FindOne(ctx, bson.M{"_id": "pageviews"}).Decode(&pageViewCounter)
	if err != nil {
		log.Println("Error getting page view counter:", err)
		pageViewCounter.Count = 0
	}

	// Get total clicks counter
	var totalClicksCounter Counter
	err = countersCollection.FindOne(ctx, bson.M{"_id": "totalClicks"}).Decode(&totalClicksCounter)
	if err != nil {
		log.Println("Error getting total clicks counter:", err)
		totalClicksCounter.Count = 0
	}

	// Get quotes
	quotesCollection := db.Collection("quotes")
	cursor, err := quotesCollection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}))
	quotes := []Quote{}
	if err == nil {
		defer cursor.Close(ctx)
		cursor.All(ctx, &quotes)
	}

	// Get GitHub repos
	repos := getGitHubRepos("wsoule")

	// Render template
	data := PageData{
		Name:          "Wyat",
		WebhookCount:  webhookCounter.Count,
		PageViewCount: pageViewCounter.Count,
		TotalClicks:   totalClicksCounter.Count,
		Quotes:        quotes,
		GitHubRepos:   repos,
	}

	err = templates.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// getGitHubRepos fetches repositories for a given GitHub username
func getGitHubRepos(username string) []GitHubRepo {
	url := fmt.Sprintf("https://api.github.com/users/%s/repos?sort=updated&per_page=100", username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("Error creating GitHub request:", err)
		return []GitHubRepo{}
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error fetching GitHub repos:", err)
		return []GitHubRepo{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("GitHub API returned status %d\n", resp.StatusCode)
		return []GitHubRepo{}
	}

	var repos []GitHubRepo
	err = json.NewDecoder(resp.Body).Decode(&repos)
	if err != nil {
		log.Println("Error decoding GitHub response:", err)
		return []GitHubRepo{}
	}

	return repos
}
