package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"greenlight.tomcat.net/internal/data"
	"greenlight.tomcat.net/internal/validator"
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

// authenticate is a middleware that handles user authentication based on the "Authorization" header.
// It performs the following steps:
// 1. Adds a "Vary: Authorization" header to the response to indicate that responses may vary based on the Authorization header.
// 2. Retrieves the "Authorization" header from the request.
// 3. If the header is empty, it sets the user in the request context to AnonymousUser and proceeds to the next handler.
// 4. If the header is present, it expects a "Bearer <token>" format.
// 5. Validates the token format and returns an invalidAuthenticationTokenResponse if the format is incorrect.
// 6. Validates the token using ValidateTokenPlaintext and returns an invalidAuthenticationTokenResponse if the token is invalid.
// 7. Retrieves the user associated with the token using GetForToken.
// 8. If the user is not found, it returns an invalidAuthenticationTokenResponse.
// 9. If any other error occurs during token retrieval, it returns a serverErrorResponse.
// 10. If the user is successfully retrieved, it sets the user in the request context and proceeds to the next handler.
func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add a "Vary: Authorization" header to the response. This indicates to any
		// caches that the response may vary based on the value of the Authorization
		// header in the request.
		w.Header().Add("Vary", "Authorization")

		// Retrieve the value of the Authorization header from the request. This will
		// usually contain the user's authentication token.
		authorizationHeader := r.Header.Get("Authorization")

		// If the Authorization header is empty, treat this as an anonymous request.
		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			// Call the next handler in the chain.
			next.ServeHTTP(w, r)
			return
		}

		// Split this into its constituent parts, and if the header isn't in the expected format
		// return 401 Unauthorized response
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// Extract the actual authentication token from the header parts
		token := headerParts[1]

		v := validator.New()

		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// Retrieve the details of the user associated with the authentication token,
		// again calling the invalidAuthenticationTokenResponse() helper if no matching record was found
		user, err := app.models.Users.GetForToken(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		// Call the contextSetUser() helper to add the user informatio to the request
		r = app.contextSetUser(r, user)

		// Call the next handler in the chain
		next.ServeHTTP(w, r)
	})
}

// requireActivatedUser is a middleware that checks if the user account is activated.
// It retrieves the user from the request context and checks if the user is activated.
// If the user is not activated, it returns an inactive account response.
// Otherwise, it calls the next handler in the chain.
// This middleware is used to protect routes that require an activated user account.
func (app *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	// Define a new http.HandlerFunc which wraps the next handler in the chain.
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Retrieve the user from the request context.
		user := app.contextGetUser(r)

		// Check if the user is not activated.
		if !user.Activated {
			// If the user is not activated, return an inactive account response.
			app.inactiveAccountResponse(w, r)
			return
		}
		// If the user is activated, call the next handler in the chain.
		next.ServeHTTP(w, r)
	})
	// Wrap the new handler with the requireAuthenticatedUser middleware to ensure the user is authenticated first.
	return app.requireAuthenticatedUser(fn)
}

// requireAuthenticatedUser is a middleware that checks if the request is from an authenticated user.
// It retrieves the user from the request context and checks if the user is anonymous.
// If the user is anonymous, it returns an authentication required response.
// Otherwise, it calls the next handler in the chain.
// This middleware is used to protect routes that require authentication.
func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

}

// requirePermission is a middleware that checks if the authenticated and activated user has a specific permission.
// It takes a permission code (string) and the next http.HandlerFunc in the chain.
// It retrieves the user from the request context, fetches their permissions from the database,
// and checks if the required permission code is included in their permissions.
// If the user does not have the required permission, it returns a 403 Forbidden response.
// If there's a database error fetching permissions, it returns a 500 Internal Server Error.
// Otherwise, it calls the next handler in the chain.
// This middleware is typically chained after requireActivatedUser to ensure the user is both
// authenticated and activated before checking permissions.
// Parameters:
// - code: The permission code (string) required to access the resource.
// - next: The next http.HandlerFunc in the middleware chain.
func (app *application) requirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		permissions, err := app.models.Permissions.GetAllForUser(user.ID)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		if !permissions.Include(code) {
			app.notPermittedResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	}

	return app.requireActivatedUser(fn)
}
