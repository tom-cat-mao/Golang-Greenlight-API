package data

import (
	"database/sql"
	"errors"
)

// ErrRecordNotFound is a sentinel error that indicates a requested database record
// could not be found. This error should be returned by data access methods when
// a query yields no results, allowing calling code to explicitly handle this case.
var ErrRecordNotFound = errors.New("record not found")

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
