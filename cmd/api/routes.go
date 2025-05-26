package main

import (
	"expvar"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// routes returns a http.Handler that serves the application's routes with middleware applied.
// It configures the router with custom error handlers and registers all application routes.
func (app *application) routes() http.Handler {
	// Initialize a new httprouter instance which implements the http.Handler interface
	router := httprouter.New()

	// Custom error handlers for the router:
	// NotFound - handles requests to undefined routes (404 errors)
	// MethodNotAllowed - handles requests with unsupported methods (405 errors)
	// Both handlers return JSON-formatted error responses consistent with our API design
	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	// Register all application routes with their corresponding HTTP methods and handlers.
	// The routes follow RESTful conventions and are versioned under /v1/ prefix.
	// Each route is documented with its purpose and functionality:

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	// GET /v1/movies - Retrieves a list of movies, applying the requireActivatedUser middleware
	// to ensure only activated users can access this resource.
	router.HandlerFunc(http.MethodGet, "/v1/movies", app.requirePermission("movies:read", app.listMoviesHandler))

	// POST /v1/movies - Creates a new movie, applying the requireActivatedUser middleware
	// to ensure only activated users can access this resource.
	router.HandlerFunc(http.MethodPost, "/v1/movies", app.requirePermission("movies:write", app.createMovieHandler))

	// GET /v1/movies/:id - Retrieves a specific movie by ID, applying the requireActivatedUser middleware.
	router.HandlerFunc(http.MethodGet, "/v1/movies/:id", app.requirePermission("movies:read", app.showMovieHandler))

	// PATCH /v1/movies/:id - Updates a specific movie by ID, applying the requireActivatedUser middleware.
	router.HandlerFunc(http.MethodPatch, "/v1/movies/:id", app.requirePermission("movies:write", app.updateMovieHandler))

	// DELETE /v1/movies/:id - Deletes a specific movie by ID, applying the requireActivatedUser middleware.
	router.HandlerFunc(http.MethodDelete, "/v1/movies/:id", app.requirePermission("movies:write", app.deleteMovieHandler))

	// POST /v1/users - Registers a new user account
	// Requires name, email and password in request body
	// Validates input and returns 201 Created on success
	// Returns 400 Bad Request for invalid data or 409 Conflict for duplicate email
	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)

	// PUT /v1/users/activated - Activates a registered user account
	// Requires a valid activation token in the request body, typically sent via email
	// On success, it updates the user's status to 'activated' and returns 200 OK with user details
	// If the token is invalid or expired, it returns 400 Bad Request with an appropriate message
	// If the token is not found, which could indicate it was already used or never existed, it returns 404 Not Found
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)

	// POST /v1/tokens/authentication - Creates a new authentication token for a user
	// Requires valid user credentials (email and password) in the request body
	// On success, it returns a new authentication token that can be used to access protected resources
	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)

	router.Handler(http.MethodGet, "/debug/vars", expvar.Handler())

	// Wrap the router with the following middleware:
	// 1. recoverPanic: Gracefully handles panics to prevent server crashes and return controlled responses.
	// 2. enableCORS: Adds CORS headers to responses.
	// 3. rateLimit: Implements rate limiting to prevent abuse and ensure fair usage.
	// 4. authenticate: Handles user authentication based on the "Authorization" header.
	// 5. metrics: Collects and publishes application metrics.
	return app.metrics(app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(router)))))
}
