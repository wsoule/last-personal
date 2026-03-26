package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yuin/goldmark"
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
	Archived        bool   `json:"archived"`
}

// BasePageData contains common fields for all page templates
type BasePageData struct {
	CurrentPage string
}

// PageData represents the data passed to the home page template
type PageData struct {
	BasePageData
	Name          string
	WebhookCount  int
	PageViewCount int
	TotalClicks   int
	Quotes        []Quote
	GitHubRepos   []GitHubRepo
}

// BlogInfo represents a blog post listing
type BlogInfo struct {
	Title    string
	Filename string
}

// Project represents a showcase project with optional live preview
type Project struct {
	Name        string
	Description string
	Tech        []string
	GitHubURL   string
	LiveURL     string
	IframeURL   string // empty means no live preview
}

// ProjectsPageData represents data passed to the projects page
type ProjectsPageData struct {
	BasePageData
	Projects    []Project
	GitHubRepos []GitHubRepo
}

// LifePageData represents data passed to the life/personal page
type LifePageData struct {
	BasePageData
}

// BlogsPageData represents the data passed to the blogs listing page
type BlogsPageData struct {
	BasePageData
	Blogs []BlogInfo
}

// BlogPageData represents the data passed to individual blog pages
type BlogPageData struct {
	BasePageData
	Title   string
	Content template.HTML
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

	// Parse templates (main templates, partials, and finance)
	templates = template.Must(template.ParseGlob("templates/*.html"))
	template.Must(templates.ParseGlob("templates/partials/*.html"))
	template.Must(templates.ParseGlob("templates/finance/*.html"))

	// Initialize and start WebSocket hub
	hub = NewHub()
	go hub.Run()

	// Routes
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/increment", incrementHandler)
	http.HandleFunc("/decrement", decrementHandler)
	http.HandleFunc("/quote", rateLimitMiddleware(quoteHandler, 5)) // 5 requests per minute
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/blogs", blogsHandler)
	http.HandleFunc("/blog/", blogHandler)
	http.HandleFunc("/projects", projectsHandler)
	http.HandleFunc("/life", lifeHandler)

	// Finance calculator pages
	http.HandleFunc("/finance", financeIndexHandler)
	http.HandleFunc("/finance/investment", financeInvestmentHandler)
	http.HandleFunc("/finance/return", financeReturnHandler)
	http.HandleFunc("/finance/loan", financeLoanHandler)
	http.HandleFunc("/finance/savings-goal", financeSavingsGoalHandler)
	http.HandleFunc("/finance/fire", financeFIREHandler)
	http.HandleFunc("/finance/inflation", financeInflationHandler)
	http.HandleFunc("/finance/debt", financeDebtHandler)

	// Finance calculator APIs
	http.HandleFunc("/api/investment-calc", rateLimitMiddleware(investmentCalcHandler, 10))
	http.HandleFunc("/api/return-calc", returnCalcHandler)
	http.HandleFunc("/api/loan-comparison", loanComparisonHandler)
	http.HandleFunc("/api/savings-goal", savingsGoalHandler)
	http.HandleFunc("/api/fire-calc", fireCalcHandler)
	http.HandleFunc("/api/inflation-calc", inflationCalcHandler)
	http.HandleFunc("/api/debt-calc", debtCalcHandler)
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

	// Render template
	data := PageData{
		BasePageData:  BasePageData{CurrentPage: "home"},
		Name:          "Wyat",
		WebhookCount:  webhookCounter.Count,
		PageViewCount: pageViewCounter.Count,
		TotalClicks:   totalClicksCounter.Count,
		Quotes:        quotes,
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

	var allRepos []GitHubRepo
	err = json.NewDecoder(resp.Body).Decode(&allRepos)
	if err != nil {
		log.Println("Error decoding GitHub response:", err)
		return []GitHubRepo{}
	}

	// Filter out archived repositories
	var repos []GitHubRepo
	for _, repo := range allRepos {
		if !repo.Archived {
			repos = append(repos, repo)
		}
	}

	return repos
}

// blogsHandler lists all blog posts
func blogsHandler(w http.ResponseWriter, r *http.Request) {
	files, err := filepath.Glob("blogs/*.md")
	if err != nil {
		http.Error(w, "Error reading blogs", http.StatusInternalServerError)
		return
	}

	var blogs []BlogInfo
	for _, file := range files {
		// Read first line as title
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		title := "Untitled"
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#") {
				title = strings.TrimSpace(strings.TrimPrefix(line, "#"))
				break
			}
		}

		filename := strings.TrimSuffix(filepath.Base(file), ".md")
		blogs = append(blogs, BlogInfo{
			Title:    title,
			Filename: filename,
		})
	}

	data := BlogsPageData{
		BasePageData: BasePageData{CurrentPage: "blogs"},
		Blogs:        blogs,
	}

	err = templates.ExecuteTemplate(w, "blogs.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// projectsHandler renders the projects showcase page
func projectsHandler(w http.ResponseWriter, r *http.Request) {
	projects := []Project{
		{
			Name:        "ngmi",
			Description: "Global leaderboards for GitHub PR review time. Search any public repo, user, or org — see avg/fastest/slowest merge times, top reviewers, and recent PRs. BFS crawler continuously expands coverage.",
			Tech:        []string{"Go", "PostgreSQL", "Redis", "HTMX"},
			LiveURL:     "https://ngmi.review",
			IframeURL:   "https://ngmi.review",
		},
		{
			Name:        "Hydrogen",
			Description: "Context-first PR review tool. Integrates with GitHub via a GitHub App — syncs PR state via webhooks, computes deterministic review slices, and renders a custom diff viewer. No code stored.",
			Tech:        []string{"Go", "SvelteKit", "PostgreSQL", "Redis"},
			LiveURL:     "https://hydrogenpr.dev",
			IframeURL:   "https://hydrogenpr.dev",
		},
		{
			Name:        "Audioparrot",
			Description: "Professional audiobook production platform. Multi-provider TTS (OpenAI, ElevenLabs, Google, local Piper), per-character voice profiles, async chapter processing, M4B export with chapter markers.",
			Tech:        []string{"Python", "RabbitMQ", "Redis", "FastAPI"},
			GitHubURL:   "",
			LiveURL:     "",
			IframeURL:   "",
		},
		{
			Name:        "Query",
			Description: "Desktop SQL client built with Tauri and React. Monaco editor with IntelliSense (tables, columns, keywords), schema browser, query history, saved queries, command palette, and OS keychain for passwords.",
			Tech:        []string{"Tauri", "Rust", "React", "TypeScript"},
			GitHubURL:   "https://github.com/brass-raven/Query",
			LiveURL:     "",
			IframeURL:   "",
		},
	}

	data := ProjectsPageData{
		BasePageData: BasePageData{CurrentPage: "projects"},
		Projects:     projects,
		GitHubRepos:  getGitHubRepos("wsoule"),
	}

	err := templates.ExecuteTemplate(w, "projects.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// lifeHandler renders the personal/life page
func lifeHandler(w http.ResponseWriter, r *http.Request) {
	data := LifePageData{
		BasePageData: BasePageData{CurrentPage: "life"},
	}

	err := templates.ExecuteTemplate(w, "life.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// blogHandler displays an individual blog post
func blogHandler(w http.ResponseWriter, r *http.Request) {
	// Extract filename from URL
	filename := strings.TrimPrefix(r.URL.Path, "/blog/")
	if filename == "" {
		http.NotFound(w, r)
		return
	}

	// Read markdown file
	filepath := filepath.Join("blogs", filename+".md")
	content, err := os.ReadFile(filepath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Parse title from first heading
	lines := strings.Split(string(content), "\n")
	title := "Blog Post"
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			title = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			break
		}
	}

	// Convert markdown to HTML
	var buf bytes.Buffer
	if err := goldmark.Convert(content, &buf); err != nil {
		http.Error(w, "Error converting markdown", http.StatusInternalServerError)
		return
	}

	data := BlogPageData{
		BasePageData: BasePageData{CurrentPage: "blogs"},
		Title:        title,
		Content:      template.HTML(buf.String()),
	}

	err = templates.ExecuteTemplate(w, "blog.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
