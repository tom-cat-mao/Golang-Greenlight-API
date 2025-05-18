package main

import (
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

	// Healthcheck endpoint - used for service monitoring and uptime checks
	// Returns application status information in JSON format
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	// GET /v1/movies - Retrieves a paginated list of all movies
	// Supports filtering by title, genres, and year range via query parameters
	// Returns movies sorted by ID in ascending order by default
	router.HandlerFunc(http.MethodGet, "/v1/movies", app.listMoviesHandler)

	// Movie resource endpoints:
	// POST /v1/movies - Creates a new movie record from JSON payload
	// Requires title, year, runtime and genres in request body
	router.HandlerFunc(http.MethodPost, "/v1/movies", app.createMovieHandler)

	// GET /v1/movies/:id - Retrieves a specific movie by its ID
	// Returns 404 if movie doesn't exist or ID is invalid
	router.HandlerFunc(http.MethodGet, "/v1/movies/:id", app.showMovieHandler)

	// PATCH /v1/movies/:id - Partially updates an existing movie record
	// Accepts partial updates - only fields provided in request body will be updated
	// Validates input data and returns appropriate error responses
	router.HandlerFunc(http.MethodPatch, "/v1/movies/:id", app.updateMovieHandler)

	// DELETE /v1/movies/:id - Deletes a movie by its ID
	// Expects a valid movie ID in the URL path
	// Returns 404 if the movie does not exist, or 200 with a success message if deleted
	router.HandlerFunc(http.MethodDelete, "/v1/movies/:id", app.deleteMovieHandler)

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

	// Wrap the router with the rate limiting middleware to control request rate
	// then wrap with the panic recovery middleware to gracefully handle panics.
	// This ensures all requests are subject to rate limiting and that any panics
	// are caught and handled with a proper error response.
	return app.recoverPanic(app.rateLimit(router))
}
