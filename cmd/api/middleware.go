package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// recoverPanic is a middleware that gracefully handles panics in the application.
// It wraps the next handler in a deferred function that catches any panics,
// ensures the connection is closed, and returns a 500 Internal Server Error response
// to the client with a generic error message.
// This prevents the server from crashing and provides a controlled response to the client.
func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Defer a function to catch any panics that occur during request processing
		defer func() {
			// Recover from any panic and convert the recovered value to an error
			if err := recover(); err != nil {
				// Set the Connection header to "close" to ensure the client knows
				// the connection will be terminated after the response
				w.Header().Set("Connection", "close")

				// Send a 500 Internal Server Error response with the recovered error
				// converted to a string. The actual error details are logged but not
				// exposed to the client for security reasons.
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		// Call the next handler in the chain
		next.ServeHTTP(w, r)
	})
}

// rateLimit is a middleware that implements rate limiting for incoming requests.
// It maintains a map of client IP addresses to track request rates and enforces
// a limit of 2 requests per second with a burst capacity of 4 requests.
func (app *application) rateLimit(next http.Handler) http.Handler {
	// client represents a rate-limited client with their limiter and last seen timestamp
	type client struct {
		limiter  *rate.Limiter // Token bucket rate limiter for this client
		lastSeen time.Time     // Last time this client made a request
	}

	var (
		mu      sync.Mutex                 // Mutex to protect concurrent access to the clients map
		clients = make(map[string]*client) // Map of client IPs to their rate limiting data
	)

	// Start a background goroutine to clean up old client entries
	go func() {
		// Run cleanup every minute
		for {
			time.Sleep(time.Minute)

			mu.Lock() // Lock the mutex for map access

			// Remove clients that haven't been seen in the last 3 minutes
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}

			mu.Unlock() // Unlock when done
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract the client IP from the request
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		mu.Lock() // Lock for client map access

		// Create a new rate limiter for new clients
		if _, found := clients[ip]; !found {
			// 2 requests per second with a burst of 4
			clients[ip] = &client{limiter: rate.NewLimiter(2, 4)}
		}

		// Update the last seen time for this client
		clients[ip].lastSeen = time.Now()

		// Check if request is allowed by rate limiter
		if !clients[ip].limiter.Allow() {
			mu.Unlock() // Unlock before returning
			app.rateLimitExceededResponse(w, r)
			return
		}

		mu.Unlock() // Unlock when done

		// If rate limit not exceeded, call the next handler
		next.ServeHTTP(w, r)
	})
}
