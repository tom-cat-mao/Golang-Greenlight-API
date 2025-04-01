package main

import (
	"net/http"
)

// healthcheckHandler returns the application status and system information in a JSON response.
// The response envelope contains:
//   - "status": String indicating service availability ("available")
//   - "system_info": System metadata including:
//     environment: deployment environment (from app.config.env)
//     version: application version (from compile-time version constant)
//
// On JSON encoding errors, logs the error using the application logger and returns a 500 status
// with a generic error message to prevent leaking sensitive error details.
func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	// Map to hold the information that we want to send in the response
	env := envelope{
		"status": "available",
		"system_info": map[string]string{
			"environment": app.config.env,
			"version":     version,
		},
	}

	// Attempt to write JSON response using the application's helper method
	err := app.writeJSON(w, http.StatusOK, env, nil)
	if err != nil {
		// Log the error and return a generic error message to the client
		app.logger.Error(err.Error())
		http.Error(w, "The server encountered a problem and could not process your request", http.StatusInternalServerError)
	}
}
