package main

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

var (
	limiters = make(map[string]*rate.Limiter)
	mu       sync.Mutex
)

// getIPAddress extracts the real IP address from the request
func getIPAddress(r *http.Request) string {
	// Check X-Forwarded-For header (used by proxies/load balancers)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if ip, _, err := net.SplitHostPort(forwarded); err == nil {
			return ip
		}
		return forwarded
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// getLimiter returns a rate limiter for the given IP and limit
func getLimiter(ip string, requestsPerMinute int) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	limiter, exists := limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(requestsPerMinute)/60, requestsPerMinute)
		limiters[ip] = limiter

		// Clean up old limiters periodically (optional, prevents memory leak)
		if len(limiters) > 10000 {
			// Simple cleanup: remove half of the limiters
			count := 0
			for k := range limiters {
				delete(limiters, k)
				count++
				if count > 5000 {
					break
				}
			}
		}
	}

	return limiter
}

// rateLimitMiddleware wraps a handler with rate limiting
func rateLimitMiddleware(next http.HandlerFunc, requestsPerMinute int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := getIPAddress(r)
		limiter := getLimiter(ip, requestsPerMinute)

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}
