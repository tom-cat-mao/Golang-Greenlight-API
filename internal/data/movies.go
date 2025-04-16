package data

import (
	"database/sql"
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

func (m MovieModel) Get(id int64) (*Movie, error) {
	return nil, nil
}

func (m MovieModel) Update(movie Movie) error {
	return nil
}

func (m MovieModel) Delete(id int64) error {
	return nil
}
