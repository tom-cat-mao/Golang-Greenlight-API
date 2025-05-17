package main

import (
	"errors"
	"net/http"

	"greenlight.tomcat.net/internal/data"
	"greenlight.tomcat.net/internal/validator"
)

// registerUserHandler handles HTTP POST requests to register new users.
// It validates the input, creates a new user record, and returns the created user.
func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	// Define an anonymous struct to hold the expected input fields from the request body
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// Read and decode the JSON request body into our input struct
	err := app.readJSON(w, r, &input)
	if err != nil {
		// If there's an error reading JSON, respond with 400 Bad Request
		app.badRequestResponse(w, r, err)
		return
	}

	// Create a new User struct with data from the request
	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false, // New users start as inactive by default
	}

	// Set the password hash from the plaintext password
	err = user.Password.Set(input.Password)
	if err != nil {
		// If password hashing fails, respond with 500 Internal Server Error
		app.serverErrorResponse(w, r, err)
		return
	}

	// Initialize a new validator instance
	v := validator.New()

	// Validate the user struct and check if validation failed
	if data.ValidateUser(v, user); !v.Valid() {
		// If validation fails, respond with 422 Unprocessable Entity
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Insert the new user record into the database
	err = app.models.Users.Insert(user)
	if err != nil {
		switch {
		// Handle case where email already exists
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		// For all other errors, respond with 500 Internal Server Error
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Call the background helper to send the welcome email
	// in the background
	app.background(func() {
		err = app.mailer.Send(user.Email, "user_welcome.html", user)
		if err != nil {
			app.logger.Error(err.Error())
		}
	})

	// Response with status 202 Accepted codde
	// indicates that the requests has beed accepted for processing
	// but the processing has not been completed
	err = app.writeJSON(w, http.StatusAccepted, envelope{"user": user}, nil)
	if err != nil {
		// If JSON writing fails, respond with 500 Internal Server Error
		app.serverErrorResponse(w, r, err)
	}
}
