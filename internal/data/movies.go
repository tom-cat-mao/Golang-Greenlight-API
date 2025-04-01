package data

import "time"

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
