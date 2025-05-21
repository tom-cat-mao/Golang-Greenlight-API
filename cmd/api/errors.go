package main

import (
	"fmt"
	"net/http"
)

// logError logs error details including HTTP method and URI from the request.
// It extracts the request method and URI, then logs the error using the application's logger
// with these contextual values for better debugging and monitoring.
func (app *application) logError(r *http.Request, err error) {
	// Extract method and URI from the request
	var (
		method = r.Method
		uri    = r.URL.RequestURI()
	)

	// Log error with extracted request details using structured logging
	app.logger.Error(err.Error(), "method", method, "uri", uri)
}

// errorResponse sends a JSON-formatted error message with the given status code.
// It accepts:
// - w: http.ResponseWriter to write the response
// - r: *http.Request for request context logging
// - status: HTTP status code to send
// - message: error message or data to send in the response (can be any type)
func (app *application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message any) {
	// Wrap the message in an envelope with "error" key for consistent JSON structure
	env := envelope{"error": message}

	// Write JSON response using application helper. Pass nil for headers since we don't need
	// to set any custom headers in this error response case.
	err := app.writeJSON(w, status, env, nil)
	if err != nil {
		// If JSON writing fails, log the error and fall back to plain text response
		app.logError(r, err)
		w.WriteHeader(500) // Send generic server error status code
	}
}

// serverErrorResponse logs the provided error and sends a 500 Internal Server Error
// response with a generic error message to the client. This is used when the server
// encounters an unexpected issue that prevents it from fulfilling the request.
// Parameters:
// - w: http.ResponseWriter to write the HTTP response
// - r: *http.Request to extract request context for logging
// - err: error that occurred, to be logged
func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	// Log the error details including request method and URI
	app.logError(r, err)

	// Create a generic user-facing error message that doesn't expose internal details
	message := "the server encountered a problem and could not process your request"

	// Send JSON error response with 500 status code using the application's errorResponse helper
	app.errorResponse(w, r, http.StatusInternalServerError, message)
}

// notFoundResponse sends a JSON-formatted 404 Not Found response to the client.
// It's used when a requested resource doesn't exist in the system.
// Parameters:
// - w: http.ResponseWriter to write the HTTP response
// - r: *http.Request to extract request context for logging
func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	// Define a user-friendly error message for the 404 response
	message := "the requested resource could not be found"

	// Use the application's errorResponse helper to send the JSON response
	// with the appropriate HTTP status code
	app.errorResponse(w, r, http.StatusNotFound, message)
}

// methodNotAllowedResponse sends a JSON-formatted 405 Method Not Allowed response to the client.
// It's used when a request method is not supported for the requested resource.
// Parameters:
// - w: http.ResponseWriter to write the HTTP response
// - r: *http.Request to extract the request method for the error message
func (app *application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	// Create a descriptive error message that includes the unsupported HTTP method
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)

	// Use the application's errorResponse helper to send the JSON response
	// with the appropriate HTTP status code
	app.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}

// badRequestResponse sends a JSON-formatted 400 Bad Request response to the client.
// It's used when the client sends invalid or malformed data in the request.
// Parameters:
// - w: http.ResponseWriter to write the HTTP response
// - r: *http.Request to extract request context for logging
// - err: error containing details about what made the request invalid
func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	// Use the application's errorResponse helper to send the JSON response
	// with a 400 status code and the error message from the provided error
	app.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

// failedValidationResponse sends a JSON-formatted 422 Unprocessable Entity response to the client.
// It's specifically used when the client's request data fails validation checks.
// The 'errors' parameter should be a map where keys are field names and values are error messages.
// Parameters:
//   - w: http.ResponseWriter to write the HTTP response.
//   - r: *http.Request to extract request context for logging.
//   - errors: A map of validation errors, where keys are field names and values are error messages.
func (app *application) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	// Use the application's errorResponse helper to send the JSON response
	// with a 422 status code and the validation errors.
	app.errorResponse(w, r, http.StatusUnprocessableEntity, errors)
}

// editConflictResponse sends a JSON-formatted 409 Conflict response to the client.
// It's used when an optimistic concurrency control check fails during a record update,
// indicating the record was modified by another process since it was fetched.
// Parameters:
//   - w: http.ResponseWriter to write the HTTP response
//   - r: *http.Request to extract request context for logging
func (app *application) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	// Define a user-friendly error message explaining the edit conflict
	message := "unable to update the record due to an edit conflict, please try again"

	// Use the application's errorResponse helper to send the JSON response
	// with HTTP 409 Conflict status code and the error message
	app.errorResponse(w, r, http.StatusConflict, message)
}

// rateLimitExceededResponse sends a JSON-formatted 429 Too Many Requests response to the client.
// This is used when the client has exceeded the allowed rate limit for requests.
// Parameters:
//   - w: http.ResponseWriter to write the HTTP response.
//   - r: *http.Request to extract request context for logging.
func (app *application) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	// Define a user-friendly error message indicating the rate limit has been exceeded.
	message := "rate limit exceeded"
	// Use the application's errorResponse helper to send the JSON response
	// with HTTP 429 Too Many Requests status code and the error message.
	app.errorResponse(w, r, http.StatusTooManyRequests, message)
}

// invalidCredentialsResponse sends a JSON-formatted 401 Unauthorized response to the client.
// It's used when the client provides invalid authentication credentials.
// Parameters:
//   - w: http.ResponseWriter to write the HTTP response.
//   - r: *http.Request to extract request context for logging.
func (app *application) invalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	message := "invalid authentication credentials"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

// invalidAuthenticationTokenResponse sends a JSON-formatted 401 Unauthorized response to the client.
// It's used when the client provides invalid or missing authentication token.
// Parameters:
//   - w: http.ResponseWriter to write the HTTP response.
//   - r: *http.Request to extract request context for logging.
func (app *application) invalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	message := "invalid or missing authentication token"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

// authenticationRequiredResponse sends a JSON-formatted 401 Unauthorized response to the client.
// It's used when the client attempts to access a protected resource without being authenticated.
// Parameters:
//   - w: http.ResponseWriter to write the HTTP response.
//   - r: *http.Request to extract request context for logging.
func (app *application) authenticationRequiredResponse(w http.ResponseWriter, r *http.Request) {
	message := "you must be authenticated to access this resource"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

// authenticationRequiredResponse sends a JSON-formatted 401 Unauthorized response to the client.
// It's used when the client attempts to access a protected resource without being authenticated.
// Parameters:
//   - w: http.ResponseWriter to write the HTTP response.
//   - r: *http.Request to extract request context for logging.
func (app *application) inactiveAccountResponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account must be activated to access this resource"
	app.errorResponse(w, r, http.StatusForbidden, message)
}
