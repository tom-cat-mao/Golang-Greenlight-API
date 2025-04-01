package main

import (
	"net/http"
)

// healthcheckHandler is an HTTP handler that returns the current status of the application.
// It responds with a JSON object containing:
// - The service status ("available")
// - The current environment (from app.config)
// - The application version (from version constant)
// If JSON serialization fails, it logs the error and returns a 500 Internal Server Error response.
func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	// Map to hold the information that we want to send in the response
	data := map[string]string{
		"status":      "available",
		"environment": app.config.env,
		"version":     version,
	}

	// Attempt to write JSON response using the application's helper method
	err := app.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		// Log the error and return a generic error message to the client
		app.logger.Error(err.Error())
		http.Error(w, "The server encountered a problem and could not process your request", http.StatusInternalServerError)
	}
}
