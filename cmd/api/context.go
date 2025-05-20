package main

import (
	"context"
	"net/http"

	"greenlight.tomcat.net/internal/data"
)

// Custom contextKey type
// with the underlying type string.
type contextKey string

// Convert the string "user" to a contextKey type and assign it to the userContextKey
// constant. Use this constant as the key for getting and setting user information
// in the request context.
const userContextKey = contextKey("user")

// returns a new copy of the request with the provided
// User struct added to the context.
// .user the userContextKey constant as the key
func (app *application) contextSetUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

// retrieves the User struct from the request context
// if it doesn't exist it will firmly be an 'unexpected' error
func (app *application) contextGetUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(userContextKey).(data.User)
	if !ok {
		panic("missing user value in request context")
	}

	return &user
}
