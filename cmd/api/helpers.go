package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// envelope is a helper type for wrapping JSON responses in a consistent structure.
// It acts as a map where string keys can map to any value type (any), allowing flexible
// response structures while maintaining a standardized format. Typically used to return:
// - A top-level "status" field indicating operation outcome
// - System information or resource data in nested objects/collections
// Example: {"status": "success", "data": { ... }}
type envelope map[string]any

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
func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	// Marshal the data to JSON, returning error if conversion fails
	js, err := json.MarshalIndent(data, "", "\t")
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

// readJSON decodes the JSON body of an HTTP request into the provided destination struct.
// It performs comprehensive error handling for various JSON-related issues, including:
// - Syntax errors in the JSON structure
// - Malformed JSON (unexpected EOF)
// - Type mismatches between JSON and struct fields
// - Unknown fields in the JSON body
// - Empty request bodies
// - Request bodies exceeding size limits
// - Invalid unmarshal targets (developer errors)
// - Multiple JSON values in request body
// The function also enforces a maximum body size of 1MB and disallows unknown fields.
func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// Limit request body size to 1MB to prevent resource exhaustion
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	// Create JSON decoder and configure to reject unknown fields
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	// Attempt to decode JSON into destination struct
	err := dec.Decode(dst)
	if err != nil {
		// Declare error type pointers for specific error handling
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		// Handle specific JSON decoding error cases
		switch {
		// Syntax error in JSON (e.g., missing comma, incorrect brackets)
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		// Unexpected EOF indicates malformed JSON structure
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		// Type mismatch error for a specific field in destination struct
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

		// Empty request body error
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		// Unknown field in JSON body
		case strings.HasPrefix(err.Error(), "json:unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		// Request body exceeds size limit
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)

		// Invalid unmarshal target (indicates programmer error)
		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		// Fallback for unexpected errors
		default:
			return err
		}
	}

	// Ensure request body contains only a single JSON value
	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errors.New("body must only contain a single JSON value")
	}

	// Return nil when decoding is successful
	return nil
}
