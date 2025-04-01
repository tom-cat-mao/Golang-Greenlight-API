package main

import (
	"fmt"
	"net/http"
	"time"

	"greenlight.tomcat.net/internal/data"
)

// createMovieHandler for the "POST /v1/movies" endpoint
func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "create a new movie")
}

// showMovieHandler handles GET requests to retrieve a movie by ID at /v1/movies/:id
// - Parses and validates the :id parameter from the URL path
// - Returns a JSON response containing mock movie data for demonstration purposes
// - Uses writeJSON helper for consistent response formatting and error handling
func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {
	// Get the value of the "id" parameters from the slice.
	id, err := app.readIDParam(r)
	if err != nil || id < 1 {
		http.NotFound(w, r)
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
		app.logger.Error(err.Error())
		http.Error(w, "The server encountered a problem and could not process your request", http.StatusInternalServerError)
	}
}
