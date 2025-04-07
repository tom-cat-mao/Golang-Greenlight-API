package main

import (
	"fmt"
	"net/http"
	"time"

	"greenlight.tomcat.net/internal/data"
	"greenlight.tomcat.net/internal/validator"
)

// createMovieHandler handles HTTP POST requests to the "/v1/movies" endpoint for creating new movie records.
// It expects a JSON payload in the request body containing the movie's title, year, runtime, and genres.
// This handler performs the following actions:
//  1. Reads and unmarshals the JSON request body into an input struct.
//  2. Validates the input data against predefined rules (e.g., required fields, data types, ranges).
//  3. If validation fails, it returns a 422 Unprocessable Entity response with detailed error messages.
//  4. If validation succeeds, it currently prints the validated input to the response body (this would be replaced with database insertion logic in a complete implementation).
//  5. Handles potential errors during JSON reading and validation, returning appropriate HTTP error responses.
func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title   string       `json:"title"`
		Year    int32        `json:"year"`
		Runtime data.Runtime `json:"runtime"`
		Genres  []string     `json:"genres"`
	}

	// Attempt to read and decode the JSON request body into the input struct.
	err := app.readJSON(w, r, &input)
	if err != nil {
		// If there's an error during JSON decoding, respond with a 400 Bad Request.
		app.badRequestResponse(w, r, err)
		return
	}

	movie := &data.Movie{
		Title:   input.Title,
		Year:    input.Year,
		Runtime: input.Runtime,
		Genres:  input.Genres,
	}

	// Create a new validator instance.
	v := validator.New()

	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// For now, just print the validated input. In a real application, this would be where you insert the data into the database.
	fmt.Fprintf(w, "%+v\n", input)
}

// showMovieHandler handles GET requests to retrieve a movie by ID at /v1/movies/:id
// - Parses and validates the :id parameter from the URL path
// - Returns a JSON response containing mock movie data for demonstration purposes
// - Uses writeJSON helper for consistent response formatting and error handling
func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {
	// Get the value of the "id" parameters from the slice.
	id, err := app.readIDParam(r)
	if err != nil || id < 1 {
		app.notFoundResponse(w, r)
		return
	}

	movie := data.Movie{
		ID:        id,
		CreatedAt: time.Now(),
		Title:     "Casablanca",
		Runtime:   102,
		Genres:    []string{"drama", "romance", "war"},
		Version:   1,
	}
	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
