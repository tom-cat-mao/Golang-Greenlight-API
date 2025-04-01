package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

// Retrieve the "id" URL parameter from the current request context, then convert it to
// an integer and return it.
// If the operation isn't successful, return 0 and an error.
func (app *application) readIDParam(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}

	return id, nil
}

// writeJSON is a helper method for sending JSON responses. It handles marshaling data,
// setting headers, and writing the response body. The function will:
// - Marshal the input data to JSON (returning error on failure)
// - Append a newline to make the response more readable
// - Set any provided headers from the headers map
// - Set the Content-Type header to application/json
// - Write the HTTP status code
// - Send the JSON response body
func (app *application) writeJSON(w http.ResponseWriter, status int, data any, headers http.Header) error {
	// Marshal the data to JSON, returning error if conversion fails
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Append newline to make terminal displays cleaner
	js = append(js, '\n')

	// Set any provided headers from the headers map
	for key, value := range headers {
		w.Header()[key] = value
	}

	// Set content type header first to ensure proper JSON handling
	w.Header().Set("Content-Type", "application/json")

	// Write HTTP status code to header
	w.WriteHeader(status)

	// Send the JSON body (already validated via Marshal)
	w.Write(js)

	return nil
}
