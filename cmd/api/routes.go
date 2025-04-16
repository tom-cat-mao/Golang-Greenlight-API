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

	// Movie resource endpoints:
	// POST /v1/movies - Creates a new movie record from JSON payload
	// Requires title, year, runtime and genres in request body
	router.HandlerFunc(http.MethodPost, "/v1/movies", app.createMovieHandler)

	// GET /v1/movies/:id - Retrieves a specific movie by its ID
	// Returns 404 if movie doesn't exist or ID is invalid
	router.HandlerFunc(http.MethodGet, "/v1/movies/:id", app.showMovieHandler)

	// PUT /v1/movies/:id - Fully updates an existing movie record
	// Requires all movie fields in request body
	// Uses optimistic concurrency control via version number
	router.HandlerFunc(http.MethodPut, "/v1/movies/:id", app.updateMovieHandler)

	// Wrap the router with the recoverPanic middleware which:
	// - Catches any panics that occur during request handling
	// - Returns a 500 Internal Server Error response
	// - Logs the error details for debugging
	return app.recoverPanic(router)
}
