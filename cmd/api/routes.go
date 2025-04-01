package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	// A new httprouter router instance
	router := httprouter.New()

	// Customize the router's behavior for 404 Not Found responses by using our application's
	// notFoundResponse handler which returns a JSON-formatted error response
	router.NotFound = http.HandlerFunc(app.notFoundResponse)

	// Customize the router's behavior for 405 Method Not Allowed responses by using our application's
	// methodNotAllowedResponse handler which returns a JSON-formatted error response
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	// Register the relevant methods, URL patterns and handler functions
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)
	router.HandlerFunc(http.MethodPost, "/v1/movies", app.createMovieHandler)
	router.HandlerFunc(http.MethodGet, "/v1/movies/:id", app.showMovieHandler)

	return app.recoverPanic(router)
}
