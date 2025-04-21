package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

	// Create a context with a 3-second timeout to ensure the database operation does not hang indefinitely.
	// The cancel function should be called to release resources once the operation completes.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel() // Ensure the context is cancelled to avoid resource leaks.

	// Execute the SQL insert statement and scan the generated ID, creation timestamp,
	// and version number into the corresponding fields of the provided movie struct.
	// This ensures the movie struct is updated with the database-generated values.
	return m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
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

	// Create a context with a 3-second timeout to ensure the database query does not hang indefinitely.
	// The cancel function should be called to release resources once the operation completes.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel() // Ensure the context is cancelled to avoid resource leaks.

	// Execute the SQL query with a context timeout and scan the result into the movie struct fields.
	// pq.Array is used to convert the PostgreSQL genres array into a Go slice.
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
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

	// Create a context with a 3-second timeout to ensure the update operation does not hang indefinitely.
	// The cancel function should be called to release resources once the operation completes.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel() // Ensure the context is cancelled to avoid resource leaks.

	// Execute the update query and attempt to scan the new version number into the movie struct.
	// If the update fails due to a version mismatch (i.e., another process has modified the record),
	// the query will return sql.ErrNoRows, which we translate to ErrEditConflict to signal a concurrency conflict.
	// Any other error is returned as-is.
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			// No rows updated: the record was changed by another process or does not exist.
			return ErrEditConflict
		default:
			// Return any other database error encountered.
			return err
		}
	}

	return nil
}

// Delete removes a movie record from the database by its ID.
// Returns:
//   - ErrRecordNotFound if the ID is invalid (<1) or no rows were deleted
//   - Any database error encountered during execution
func (m MovieModel) Delete(id int64) error {
	// Validate the ID; must be a positive integer
	if id < 1 {
		return ErrRecordNotFound
	}

	// SQL query to delete the movie with the specified ID
	query := `
		DELETE FROM movies
		WHERE id = $1
		`

	// Create a context with a 3-second timeout to ensure the delete operation does not hang indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	// Ensure the context is cancelled to free up resources once the operation completes.
	defer cancel()

	// Execute the SQL DELETE statement to remove the movie with the specified ID.
	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		// If an error occurs during the execution of the DELETE statement, return it.
		return err
	}

	// Check how many rows were affected (should be 1 if deleted)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		// Return any error encountered while checking affected rows
		return err
	}

	// If no rows were affected, the movie was not found
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	// Successful deletion
	return nil
}

// GetAll retrieves a list of movies from the database, optionally filtered by title and genres,
// and paginated/sorted according to the provided Filters struct.
// Parameters:
//   - title:   Filter movies by title (empty string means no filtering by title)
//   - genres:  Filter movies by genres (empty slice means no filtering by genres)
//   - filters: Pagination and sorting options (page, page_size, sort, etc.)
//
// Returns:
//   - A slice of pointers to Movie structs representing the retrieved movies
//   - An error if any occurs during the query or scanning process
func (m MovieModel) GetAll(title string, genres []string, filters Filters) ([]*Movie, error) {
	// Build the SQL query for retrieving movies with optional filtering, sorting, and pagination.
	// - The WHERE clause filters by title using full-text search (if a title is provided), or matches all if empty.
	// - The genres filter uses the @> operator to check if the movie's genres array contains all specified genres, or matches all if the genres slice is empty.
	// - The ORDER BY clause uses dynamic column and direction from the Filters struct, and always sorts by id as a secondary key for deterministic ordering.
	// - LIMIT and OFFSET are used for pagination.
	query := fmt.Sprintf(`
		SELECT id, created_at, title, year, runtime, genres, version
		FROM movies
		WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
		AND (genres @> $2 OR $2 = '{}')
		ORDER BY %s %s, id ASC
		LIMIT $3 OFFSET $4
		`, filters.sortColumn(), filters.sortDirection())
	// Create a context with a 3-second timeout to avoid hanging queries.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Prepare the arguments for the SQL query:
	// - $1: title filter for full-text search (empty string means no filtering)
	// - $2: genres filter as a Postgres array (empty array means no filtering)
	// - $3: limit for pagination (maximum number of results per page)
	// - $4: offset for pagination (number of results to skip)
	args := []any{title, pq.Array(genres), filters.limit(), filters.offset()}

	// Execute the SQL query using the constructed query string and arguments for filtering, sorting, and pagination.
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		// Return any error encountered during query execution.
		return nil, err
	}
	// Ensure the rows are closed after processing to free up database resources.
	defer rows.Close()

	// Prepare a slice to hold the resulting movies.
	movies := []*Movie{}

	// Iterate over the rows in the result set.
	for rows.Next() {
		var movie Movie

		// Scan the current row into the movie struct.
		err := rows.Scan(
			&movie.ID,
			&movie.CreatedAt,
			&movie.Title,
			&movie.Year,
			&movie.Runtime,
			pq.Array(&movie.Genres),
			&movie.Version,
		)
		if err != nil {
			return nil, err
		}

		// Append the movie to the result slice.
		movies = append(movies, &movie)
	}

	// Check for any errors encountered during iteration.
	if err = rows.Err(); err != nil {
		return nil, err
	}

	// Return the slice of movies and nil error.
	return movies, nil
}
