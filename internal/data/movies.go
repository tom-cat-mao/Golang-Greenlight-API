package data

import (
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
	"greenlight.tomcat.net/internal/validator"
)

// Movie represents a single movie in the database. It includes core details about the film
// along with metadata like creation timestamp and version number for optimistic locking.
// The struct tags control how the data appears when serialized to JSON:
// - CreatedAt is excluded from JSON output
// - Year, Runtime, and Genres are omitted from JSON if empty
// - All other fields are included in JSON output by default
type Movie struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"`
	Runtime   Runtime   `json:"runtime,omitempty"`
	Genres    []string  `json:"genres,omitempty"`
	Version   int32     `json:"version"`
}

// MovieModel wraps a sql.DB connection pool and provides methods for interacting
// with the movies table in the database. This struct serves as the data access layer
// for movie-related operations, implementing the repository pattern.
//
// Fields:
//   - DB: A pointer to a sql.DB connection pool that will be used to execute
//     database queries and commands.
type MovieModel struct {
	DB *sql.DB
}

func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "must be provided")
	v.Check(len(movie.Title) <= 500, "title", "must not be more than 500 bytes long")

	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 1888, "year", "must be greater than 1888")
	v.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be a positive integer")

	v.Check(movie.Genres != nil, "genres", "must be provided")
	v.Check(len(movie.Genres) >= 1, "genres", "must contain at least 1 genre")
	v.Check(len(movie.Genres) <= 5, "genres", "must not contain more than 5 genres")
	v.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")
}

// Insert adds a new movie record to the database and updates the movie struct with
// the generated ID, creation timestamp, and version number.
// Parameters:
//   - movie: A pointer to a Movie struct containing the movie data to insert
//
// Returns:
//   - error: Any database error that occurs during the operation
func (m MovieModel) Insert(movie *Movie) error {
	// Define the SQL query for inserting a new movie record.
	// The query includes parameters for title, year, runtime, and genres,
	// and returns the auto-generated ID, creation timestamp, and version.
	query := `
			INSERT INTO MOVIES (title, year, runtime, genres)
			VALUES ($1, $2, $3, $4)
			RETURNING id, created_at, version
		`

	// Prepare the arguments for the query, converting the genres slice to a PostgreSQL array
	args := []any{movie.Title, movie.Year, movie.Runtime, pq.Array(movie.Genres)}

	// Execute the query and scan the returned values into the movie struct
	// This populates the ID, CreatedAt, and Version fields of the movie
	return m.DB.QueryRow(query, args...).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
}

// Get retrieves a movie record from the database by its ID.
// Parameters:
//   - id: The ID of the movie to retrieve (must be a positive integer)
//
// Returns:
//   - *Movie: A pointer to a Movie struct containing the retrieved data
//   - error: Any error that occurs during the operation, including:
//   - ErrRecordNotFound if the ID doesn't exist or is invalid
//   - Database errors for other failures
func (m MovieModel) Get(id int64) (*Movie, error) {
	// Validate that the ID is positive
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	// Define the SQL query to select a movie by ID
	// The query retrieves all movie fields from the database
	query := `
		SELECT id, created_at, title, year, runtime, genres, version
		FROM movies
		WHERE id = $1
		`

	// Initialize an empty Movie struct to hold the retrieved data
	var movie Movie

	// Execute the query and scan the result into the movie struct
	// Note: pq.Array() is used to properly scan the PostgreSQL array into a Go slice
	err := m.DB.QueryRow(query, id).Scan(
		&movie.ID,
		&movie.CreatedAt,
		&movie.Title,
		&movie.Year,
		&movie.Runtime,
		pq.Array(&movie.Genres),
		&movie.Version,
	)

	// Handle any errors that occurred during the query execution
	if err != nil {
		switch {
		// If no rows were found, return our custom ErrRecordNotFound error
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		// For all other errors, return them directly
		default:
			return nil, err
		}
	}

	// Return a pointer to the populated movie struct
	return &movie, nil
}

// Update modifies an existing movie record in the database using optimistic concurrency control.
// It performs an atomic update of all movie fields and increments the version number to prevent race conditions.
// The update will only succeed if the movie's current version matches the expected version.
// Returns:
//   - error: Any error that occurs during the operation, including:
//   - ErrEditConflict if the version check fails (indicating concurrent modification)
//   - Database errors for connection/query failures
//   - sql.ErrNoRows if no record was found (though this is converted to ErrEditConflict)
func (m MovieModel) Update(movie Movie) error {
	// Define the SQL query for updating a movie record with optimistic concurrency control.
	// The query performs an atomic update that:
	// - Sets all movie fields (title, year, runtime, genres)
	// - Increments the version number to prevent race conditions
	// - Uses both ID and current version in WHERE clause to ensure:
	//   * The correct record is targeted (by ID)
	//   * The record hasn't been modified since it was fetched (by version)
	// - Returns the new version number via RETURNING clause for verification
	query := `
		UPDATE movies
		SET title = $1, year = $2, runtime = $3, genres = $4, version = version + 1
		WHERE id = $5 AND version = $6
		RETURNING version
		`

	// Prepare the arguments for the query in the correct order
	// Note: pq.Array() is used to properly handle the PostgreSQL array type for genres
	args := []any{
		movie.Title,
		movie.Year,
		movie.Runtime,
		pq.Array(movie.Genres),
		movie.ID,
		movie.Version,
	}

	// Execute the SQL query to update the movie record and scan the new version number
	err := m.DB.QueryRow(query, args...).Scan(&movie.Version)
	if err != nil {
		switch {
		// If no rows were affected, it means the version check failed (concurrent modification)
		// Return our custom ErrEditConflict to indicate an optimistic concurrency control violation
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		// For all other database errors, return them directly
		default:
			return err
		}
	}

	return nil
}

func (m MovieModel) Delete(id int64) error {
	return nil
}
