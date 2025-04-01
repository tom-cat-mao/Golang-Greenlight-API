package main

import (
	"fmt"
	"net/http"
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
