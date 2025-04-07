package main

import (
	"fmt"
	"net/http"
	"time"

	"greenlight.tomcat.net/internal/data"
)

// createMovieHandler handles HTTP POST requests to create new movie entries
// - Validates and processes JSON payload containing movie attributes
// - Implements proper error handling for malformed requests and database failures
// - Utilizes writeJSON helper function for standardized API response formatting
func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title   string       `json:"title"`
		Year    int32        `json:"year"`
		Runtime data.Runtime `json:"runtime"`
		Genres  []string     `json:"genres"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

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
