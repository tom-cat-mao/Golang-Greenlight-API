package main

import (
	"errors"
	"net/http"
	"time"

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

	// Add the "movies:read" permission for the new user.
	// This grants them the ability to read movie data.
	err = app.models.Permissions.AddForUser(user.ID, "movies:read")
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Initialize new token for the new user
	// with the expire time of 3 days
	// after the user record has been created in the database
	token, err := app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Run a background goroutine for the email sending
	// Define a map to act as a 'holding structure' for the data
	// contains the plaintext version of the activation token for the user
	// along with their ID.
	app.background(func() {
		data := map[string]any{
			"activationToken": token.Plaintext,
			"userID":          user.ID,
		}

		err = app.mailer.Send(user.Email, "user_welcome.html", data)
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

// Activate the User with the token which the client sent
func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the plaintext activation token from the request body
	var input struct {
		TokenPlaintext string `json:"token"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Validate the plaintext token provided by the client
	v := validator.New()

	if data.ValidateTokenPlaintext(v, input.TokenPlaintext); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Retrieve the details of the user associated with the token
	// If no matching record is found,
	// then we let the client know that the token
	// they provided is not valid
	user, err := app.models.Users.GetForToken(data.ScopActivation, input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("token", "invalid or expired activation token")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Update the user's activatio status
	user.Activated = true

	// Save the updated user record in our database,
	// checking for any edit conflicts
	err = app.models.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// If everything went successfully, then we delete all activation tokens for the user
	err = app.models.Tokens.DeleteAllForUser(data.ScopActivation, user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Send the updated user detals to the client in a JSON response
	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
