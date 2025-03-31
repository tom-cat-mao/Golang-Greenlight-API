package main

import (
	"fmt"
	"net/http"
)

// createMovieHandler for the "POST /v1/movies" endpoint
func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "create a new movie")
}

// showMovieHandler for the "GET /v1/movies/:id" endpoint
// - Retrieve the interppolated "id" parameter from the current URL and include it in a placeholder
// respose.
func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {
	// Get the value of the "id" parameters from the slice.
	id, err := app.readIDParam(r)
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}

	// Interpolate the movie ID ina placeholder response.
	fmt.Fprintf(w, "show the details of movie %d\n", id)
}
