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

// updateMovieHandler handles HTTP PUT/PATCH requests to update an existing movie record.
// The handler performs the following operations:
//  1. Extracts and validates the movie ID from the URL path parameters
//  2. Retrieves the existing movie record from the database
//  3. Reads and decodes the JSON request body into a partial update struct
//  4. Conditionally updates movie fields with non-nil values from the input
//  5. Validates the updated movie data using the validator
//  6. Persists the changes to the database
//  7. Returns the updated movie data as JSON with 200 OK status
//
// Error handling includes:
//   - 404 Not Found for invalid/missing IDs or non-existent movies
//   - 400 Bad Request for malformed JSON
//   - 422 Unprocessable Entity for validation failures
//   - 500 Internal Server Error for database/processing failures
func (app *application) updateMovieHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the movie ID from the URL and validate it
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Retrieve the existing movie record from the database
	movie, err := app.models.Movies.Get(id)
	if err != nil {
		switch {
		// Return 404 Not Found if the movie doesn't exist
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		// Return 500 Internal Server Error for other database errors
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Define an input struct to hold the expected data from the request body
	var input struct {
		Title   *string       `json:"title"`
		Year    *int32        `json:"year"`
		Runtime *data.Runtime `json:"runtime"`
		Genres  []string      `json:"genres"`
	}

	// Read and decode the JSON request body into the input struct
	err = app.readJSON(w, r, &input)
	if err != nil {
		// Return 400 Bad Request if the JSON is malformed
		app.badRequestResponse(w, r, err)
		return
	}

	if input.Title != nil {
		movie.Title = *input.Title
	}

	if input.Year != nil {
		movie.Year = *input.Year
	}

	if input.Runtime != nil {
		movie.Runtime = *input.Runtime
	}

	if input.Genres != nil {
		movie.Genres = input.Genres
	}

	// Initialize a new validator and validate the updated movie
	v := validator.New()
	if data.ValidateMovie(v, movie); !v.Valid() {
		// Return 422 Unprocessable Entity if validation fails
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Attempt to update the movie record in the database
	err = app.models.Movies.Update(*movie)
	if err != nil {
		switch {
		// If we get an edit conflict error (version mismatch), return a 409 Conflict response
		// This indicates the record was modified by another process since we fetched it
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		// For all other database errors, return a 500 Internal Server Error
		// This includes connection issues, query errors, etc.
		default:
			app.serverErrorResponse(w, r, err)
		}
		// Return early since we encountered an error
		return
	}

	// Write the updated movie as JSON response with 200 OK status
	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		// Return 500 Internal Server Error if JSON encoding fails
		app.serverErrorResponse(w, r, err)
	}
}

// deleteMovieHandler handles HTTP DELETE requests to remove a movie by its ID.
// It expects the movie ID as a URL parameter, deletes the movie from the database,
// and returns a confirmation message if successful.
func (app *application) deleteMovieHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the movie ID from the URL parameter.
	id, err := app.readIDParam(r)
	if err != nil {
		// If the ID is invalid or missing, respond with 404 Not Found.
		app.notFoundResponse(w, r)
		return
	}

	// Attempt to delete the movie from the database.
	err = app.models.Movies.Delete(id)
	if err != nil {
		switch {
		// If the movie does not exist, respond with 404 Not Found.
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		// For any other error, respond with 500 Internal Server Error.
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// If deletion is successful, return a JSON response with a success message.
	err = app.writeJSON(w, http.StatusOK, envelope{"message": "movie successfully deleted"}, nil)
	if err != nil {
		// If there is an error encoding the JSON response, return a 500 error.
		app.serverErrorResponse(w, r, err)
	}
}
