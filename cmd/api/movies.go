package main

import (
	"errors"
	"fmt"
	"net/http"

	"greenlight.tomcat.net/internal/data"
	"greenlight.tomcat.net/internal/validator"
)

// createMovieHandler handles HTTP POST requests to the "/v1/movies" endpoint for creating new movie records.
// It expects a JSON payload in the request body containing the movie's title, year, runtime, and genres.
// This handler performs the following actions:
//  1. Reads and decodes the JSON request body into an input struct
//  2. Validates the input data using the ValidateMovie function
//  3. If validation fails, returns a 422 Unprocessable Entity response with validation errors
//  4. If validation succeeds, inserts the movie record into the database
//  5. Returns a 201 Created response with the newly created movie data
//  6. Handles potential errors during JSON decoding, validation, and database operations
//  7. Sets the Location header to the newly created resource
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

	// Insert the validated movie data into the database using the MovieModel.
	// If the insertion fails, respond with a 500 Internal Server Error.
	err = app.models.Movies.Insert(movie)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Create a new http.Header map to store response headers
	headers := make(http.Header)
	// Set the Location header to point to the newly created movie resource
	// The URL follows the pattern /v1/movies/{id} where {id} is the movie's database ID
	headers.Set("Location", fmt.Sprintf("/v1/movies/%d", movie.ID))

	// Write the JSON response with:
	// - HTTP status code 201 (Created)
	// - The movie data wrapped in an envelope
	// - The Location header set to the new resource
	err = app.writeJSON(w, http.StatusCreated, envelope{"movie": movie}, headers)
	if err != nil {
		// If JSON encoding fails, respond with a 500 Internal Server Error
		app.serverErrorResponse(w, r, err)
	}
}

// showMovieHandler handles GET requests to retrieve a movie by ID from the database
// - Extracts and validates the ID parameter from the URL path
// - Retrieves the movie record from the database using the MovieModel
// - Handles various error cases:
//   - Invalid ID format (404 Not Found)
//   - Non-existent movie (404 Not Found)
//   - Database errors (500 Internal Server Error)
//
// - Returns a JSON response with the movie data on success
func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {
	// Get the value of the "id" parameters from the slice.
	id, err := app.readIDParam(r)
	if err != nil || id < 1 {
		app.notFoundResponse(w, r)
		return
	}

	// Retrieve the movie from the database using the provided ID
	movie, err := app.models.Movies.Get(id)
	if err != nil {
		switch {
		// If the error is ErrRecordNotFound, return a 404 Not Found response
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		// For all other errors, return a 500 Internal Server Error response
		default:
			app.serverErrorResponse(w, r, err)
		}
		// Return early since we encountered an error
		return
	}

	// Write the JSON response with:
	// - HTTP status code 200 (OK)
	// - The movie data wrapped in an envelope
	// - No additional headers (nil)
	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		// If JSON encoding fails, respond with a 500 Internal Server Error
		app.serverErrorResponse(w, r, err)
	}
}
