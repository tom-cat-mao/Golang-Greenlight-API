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

// Models is a container struct that holds all our database models.
// This provides a single access point to all data models in the application,
// making dependency injection cleaner and more maintainable.
// Fields:
//   - Movies: The movie database model for CRUD operations on movie records
//   - Users: The user database model for CRUD operations on user records
type Models struct {
	Movies MovieModel
	Users  UserModel
}

// NewModels initializes and returns a Models struct containing all database models.
// It takes a *sql.DB connection pool as input and injects it into each model,
// allowing all models to share the same database connection.
// Returns:
//   - Models: A struct containing initialized MovieModel and UserModel instances
func NewModels(db *sql.DB) Models {
	return Models{
		Movies: MovieModel{DB: db}, // Initialize movie model with database connection
		Users:  UserModel{DB: db},  // Initialize user model with database connection
	}
}
