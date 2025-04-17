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

// Models wraps all database model types in a single struct.
// This provides a convenient way to access all data operations
// through a single dependency rather than managing each model separately.
// Current models included:
//   - Movies: Handles all movie-related database operations
type Models struct {
	Movies MovieModel
}

// NewModels initializes and returns a new Models struct containing all database models.
// It takes a *sql.DB connection pool as input and injects it into each model,
// allowing them to share the same database connection.
// Returns:
//   - Models: A struct containing all initialized data models with the provided DB connection
func NewModels(db *sql.DB) Models {
	return Models{
		Movies: MovieModel{DB: db}, // Initialize MovieModel with the database connection
	}
}
