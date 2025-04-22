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
		// Check if rate limiting is enabled in the application configuration.
		if app.config.limiter.enabled {

			// Extract the client IP address from the request's RemoteAddr field.
			// RemoteAddr is in the form "IP:port", so we split it to get just the IP.
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				// If there's an error extracting the IP, respond with a server error and return.
				app.serverErrorResponse(w, r, err)
				return
			}

			// Lock the mutex before accessing or modifying the clients map to ensure thread safety.
			mu.Lock() // Lock for client map access

			// If this is a new client (IP not seen before), create a new rate limiter for them.
			if _, found := clients[ip]; !found {
				// Create a new rate limiter for this client using the configured requests per second (rps)
				// and burst values from the application config.
				clients[ip] = &client{
					limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst),
				}
			}

			// Update the lastSeen timestamp for this client to the current time.
			clients[ip].lastSeen = time.Now()

			// Check if the client's rate limiter allows this request.
			if !clients[ip].limiter.Allow() {
				// If not allowed (rate limit exceeded), unlock the mutex and send a 429 response.
				mu.Unlock() // Unlock before returning
				app.rateLimitExceededResponse(w, r)
				return
			}

			// Unlock the mutex after we're done with the clients map.
			mu.Unlock() // Unlock when done
		}

		// If rate limit not exceeded, call the next handler
		next.ServeHTTP(w, r)
	})
}
