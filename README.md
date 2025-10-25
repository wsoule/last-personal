# Personal Website

A minimalist personal website built with Go, MongoDB, and WebSockets. Inspired by [motherfuckingwebsite.com](https://motherfuckingwebsite.com/) and [justfuckingusehtml.com](https://justfuckingusehtml.com/).

## Features

- **About Section**: Headshot, bio, and professional experience
- **Real-time Webhook Counter**: WebSocket-powered counter with optimistic UI updates that syncs across all browsers instantly
- **GitHub Repositories**: Auto-fetched from GitHub API
- **Quotes System**: User-submitted quotes with local timezone display
- **Resume Download**: PDF resume link
- **Navigation**: Simple table of contents for easy page navigation
- **Rate Limiting**: Spam protection on quote submissions

## Tech Stack

- **Backend**: Go with modular architecture
- **Database**: MongoDB
- **Real-time**: WebSockets (gorilla/websocket)
- **Frontend**: Vanilla JavaScript, HTML (no frameworks)
- **Deployment**: Railway-ready

## Project Structure

```
.
├── main.go                 # Main application setup & home handler
├── counter.go              # Webhook counter feature & handlers
├── quotes.go               # Quote submission feature
├── websocket.go            # WebSocket hub for real-time updates
├── middleware.go           # Rate limiting middleware
├── templates/
│   └── index.html         # HTML template with WebSocket client
├── static/
│   ├── headshot.jpg       # Profile photo
│   └── resume.pdf         # Resume PDF
├── go.mod                 # Go dependencies
└── README.md              # This file
```

## Running Locally

1. **Install Dependencies**:
   ```bash
   go mod download
   ```

2. **Set up MongoDB** (or set the `MONGO_URI` environment variable):
   ```bash
   export MONGO_URI="mongodb://localhost:27017"
   ```

3. **Run the application**:
   ```bash
   go run .
   ```

   **Note**: Use `go run .` (not `go run main.go`) to compile all Go files together.

4. **Visit**: `http://localhost:8080`

## Deploying to Railway

1. Set up MongoDB database (Railway offers MongoDB as an add-on)

2. Set environment variables in Railway:
   - `MONGO_URI`: Your MongoDB connection string
   - `PORT`: Automatically set by Railway

3. Deploy your code to Railway

## MongoDB Collections

The application uses three collections:

- **`counters`**: Stores webhook and page view counters
  - Document with `_id: "webhook"` for webhook counter
  - Document with `_id: "pageviews"` for page view counter

- **`quotes`**: Stores user-submitted quotes with name, quote text, and timestamp

## How Real-time Updates Work

The site uses WebSockets for instant synchronization:

1. **Client connects** → WebSocket established on page load
2. **User clicks increment/decrement** → UI updates instantly (optimistic)
3. **Server updates MongoDB** → Broadcasts new count to all connected clients
4. **All browsers sync** → Everyone sees the update within milliseconds

### Optimistic Updates
- Counter updates immediately on click before server confirmation
- Pending requests tracked to prevent race conditions during lag
- Graceful error handling with automatic revert on failure

## Rate Limiting

Rate limiting is applied per IP address:

- **Quote submissions**: 5 requests per minute (prevents spam)
- **All other endpoints**: No rate limiting for optimal UX

Rate limiting works correctly with proxies/load balancers by checking `X-Forwarded-For` and `X-Real-IP` headers.

## Customization

1. **Replace headshot**: Add your photo at `static/headshot.jpg`
2. **Replace resume**: Add your PDF at `static/resume.pdf`
3. **Update experience**: Edit the About section in `templates/index.html`
4. **Change GitHub username**: Update `wsoule` in `main.go` line 140

## Dependencies

- `go.mongodb.org/mongo-driver` - MongoDB driver
- `github.com/gorilla/websocket` - WebSocket support
- `golang.org/x/time/rate` - Rate limiting

## License

Copyright © 2025 Wyat
