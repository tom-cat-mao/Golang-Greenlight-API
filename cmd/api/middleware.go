package main

import (
	"fmt"
	"net/http"

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

// rateLimit is a middleware that applies a global rate limit to all incoming HTTP requests.
// It uses a token bucket algorithm (from golang.org/x/time/rate) to allow up to 2 requests per second
// with a maximum burst of 4 requests. If the rate limit is exceeded, it responds with a 429 Too Many Requests error.
func (app *application) rateLimit(next http.Handler) http.Handler {
	// Create a new rate limiter allowing 2 requests per second with a burst of 4.
	limiter := rate.NewLimiter(2, 4)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if a request is allowed by the rate limiter.
		if !limiter.Allow() {
			// If not allowed, send a 429 Too Many Requests response and return early.
			app.rateLimitExceededResponse(w, r)
			return
		}

		// If allowed, call the next handler in the chain.
		next.ServeHTTP(w, r)
	})
}
