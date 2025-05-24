package data

import (
	"database/sql"
	"errors"
)

// ErrRecordNotFound is a sentinel error returned when a database query returns no rows.
// It enables explicit handling of "not found" cases in calling code, typically resulting
// in a 404 Not Found HTTP response in API handlers. This error should be used consistently
// across all data access methods to maintain uniform error handling behavior.
var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

// Models struct holds instances of all data models (MovieModel, UserModel, etc.).
// This allows us to group all data access objects together and pass them around
// as a single dependency.
type Models struct {
	// Movies provides methods for interacting with the 'movies' table.
	Movies MovieModel
	// Users provides methods for interacting with the 'users' table.
	Users UserModel
	// Tokens provides methods for interacting with the 'tokens' table.
	Tokens TokenModel
	// Permissions provides methods for interacting with the 'permissions' and 'users_permissions' tables.
	Permissions PermissionModel
}

// NewModels initializes and returns a Models struct containing all database models.
// It takes a *sql.DB connection pool as input and injects it into each model,
// allowing all models to share the same database connection.
// Returns:
//   - Models: A struct containing initialized MovieModel and UserModel instances
func NewModels(db *sql.DB) Models {
	return Models{
		Movies:      MovieModel{DB: db},      // Initialize movie model with database connection
		Users:       UserModel{DB: db},       // Initialize user model with database connection
		Tokens:      TokenModel{DB: db},      // Initialize tokens model with database connection
		Permissions: PermissionModel{DB: db}, // Initialize permissions model with database connection
	}
}
