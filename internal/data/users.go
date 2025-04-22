package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"greenlight.tomcat.net/internal/validator"
)

var (
	// ErrDuplicateEmail is returned when a user attempts to register with an email
	// that already exists in the database. This helps maintain email uniqueness
	// as enforced by the UNIQUE constraint in the users table.
	ErrDuplicateEmail = errors.New("duplicate email")
)

// User represents a user in the system.
type User struct {
	ID        int64     `json:"id"`         // Unique identifier for the user.
	CreatedAt time.Time `json:"created_at"` // Timestamp when the user was created.
	Name      string    `json:"name"`       // Name of the user.
	Email     string    `json:"email"`      // Email address of the user.
	Password  password  `json:"-"`          // Hashed password (not exposed in JSON).
	Activated bool      `json:"activated"`  // Indicates if the user's account is activated.
	Version   int       `json:"-"`          // Version number for optimistic concurrency control (not exposed in JSON).
}

// UserModel wraps a sql.DB connection pool and provides methods for interacting
// with the users table in the database. This follows the repository pattern,
// keeping database operations separate from business logic.
type UserModel struct {
	DB *sql.DB // Database connection pool for executing SQL queries
}

// password holds both the plaintext (for validation, if present) and the bcrypt hash of a user's password.
// The plaintext field is a pointer to a string so it can be nil when not needed (e.g., when loading from the database).
// The hash field stores the bcrypt hash of the password.
type password struct {
	plaintext *string // Plaintext password, used only for validation and never stored in the database.
	hash      []byte  // Bcrypt hash of the password.
}

// Set hashes the provided plaintext password using bcrypt and stores both the plaintext (for validation)
// and the resulting hash in the password struct. The plaintext is stored as a pointer for optional presence.
// Returns an error if hashing fails.
func (p *password) Set(plaintextPassword string) error {
	// Generate a bcrypt hash of the plaintext password with a cost of 12.
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		// Return the error if hashing fails.
		return err
	}

	// Store the plaintext password (as a pointer) for validation purposes.
	p.plaintext = &plaintextPassword
	// Store the bcrypt hash for authentication.
	p.hash = hash

	return nil
}

// Matches compares a plaintext password against the stored bcrypt hash.
// Returns true if the password matches the hash, false if it doesn't match,
// or an error if the comparison fails (other than a password mismatch).
func (p *password) Matches(plaintextPassword string) (bool, error) {
	// Compare the provided plaintext password with the stored hash
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))
	if err != nil {
		// Handle different error cases
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			// Password doesn't match hash, but this isn't an error condition
			return false, nil
		default:
			// Return any other error (e.g., malformed hash)
			return false, err
		}
	}

	// If no error, the password matches
	return true, nil
}

// ValidatePasswordPlaintext checks that a plaintext password meets basic security requirements.
// It validates that the password is not empty, is at least 8 bytes long (for security),
// and is not more than 72 bytes long (bcrypt's maximum supported length).
// The validation results are added to the provided validator instance.
func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	// Check that password is not empty
	v.Check(password != "", "password", "must be provided")
	// Check minimum length requirement (8 bytes)
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	// Check maximum length requirement (72 bytes - bcrypt limit)
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

// ValidateEmail checks that an email address meets basic format requirements.
// It validates that the email is not empty and matches a standard email regex pattern.
// The validation results are added to the provided validator instance.
func ValidateEmail(v *validator.Validator, email string) {
	// Check that email is not empty
	v.Check(email != "", "email", "must be provided")
	// Check that email matches the standard email regex pattern
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

// ValidateUser performs validation checks on a User struct and adds any validation errors to the validator.
// It checks:
// - Name is not empty and within length limits
// - Email is valid (using ValidateEmail helper)
// - Password plaintext (if provided) meets requirements (using ValidatePasswordPlaintext helper)
// - Password hash exists (panics if missing as this indicates a programming error)
func ValidateUser(v *validator.Validator, user *User) {
	// Validate name field - must be provided and not exceed 500 bytes
	v.Check(user.Name != "", "name", "must be provided")
	v.Check(len(user.Name) <= 500, "name", "must not be more than 500 bytes long")

	// Validate email using standard email validation
	ValidateEmail(v, user.Email)

	// If plaintext password is provided, validate it meets security requirements
	if user.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.plaintext)
	}

	// Ensure password hash exists - this should never be nil in normal operation
	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
}

// Insert adds a new user record to the database and updates the user struct with generated values.
// It returns an error if the operation fails, including ErrDuplicateEmail if the email already exists.
func (m UserModel) Insert(user *User) error {
	// SQL query to insert a new user and return the generated ID, creation timestamp, and version
	query := `
		INSERT INTO users (name, email, password_hash, activated)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version
		`

	// Arguments for the SQL query, extracted from the user struct
	args := []any{
		user.Name,
		user.Email,
		user.Password.hash,
		user.Activated,
	}

	// Create a context with a 3-second timeout to prevent long-running database operations
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel() // Ensure resources are released when function exits

	// Execute the query and scan the returned values into the user struct
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&user.ID, &user.CreatedAt, &user.Version)
	if err != nil {
		// Handle specific error cases
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			// Return custom error for duplicate email violation
			return ErrDuplicateEmail
		default:
			// Return any other database error
			return err
		}
	}

	// Return nil if the operation completed successfully
	return nil
}

// GetByEmail retrieves a user record from the database by email address.
// It returns a pointer to a User struct if found, or ErrRecordNotFound if no matching record exists.
// Any other database errors are returned as-is.
func (m UserModel) GetByEmail(email string) (*User, error) {
	// SQL query to select user fields by email
	query := `
		SELECT id, created_id, name, email, password_hash, activated, version
		FROM users
		WHERE email = $1
		`

	// Initialize an empty User struct to hold the result
	var user User

	// Create a context with a 3-second timeout to prevent long-running database operations
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel() // Ensure resources are released when function exits

	// Execute the query and scan the result into the User struct fields
	err := m.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
	)

	// Handle any errors that occurred during query execution
	if err != nil {
		switch {
		// Special case: return custom error when no matching record is found
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		// For all other errors, return them directly
		default:
			return nil, err
		}
	}

	// Return the populated user struct if no errors occurred
	return &user, nil
}

// Update modifies a user record in the database. It updates all fields except ID and CreatedAt,
// and implements optimistic concurrency control using the version field.
// Returns ErrDuplicateEmail if the email already exists, ErrEditConflict if the version doesn't match,
// or other database errors as-is.
func (m UserModel) Update(user *User) error {
	// SQL query to update user fields and increment version number.
	// The WHERE clause ensures we only update if the version matches (optimistic locking).
	// RETURNING clause gives us the new version number.
	query := `
		UPDATE users
		SET name = $1, email = $2, password_hash = $3, activated = $4, version = version + 1
		WHERE id = $5 AND version = $6
		RETURNING version
		`

	// Prepare arguments for the query in the correct order
	args := []any{
		user.Name,
		user.Email,
		user.Password.hash,
		user.Activated,
		user.ID,
		user.Version,
	}

	// Create a context with a 3-second timeout to prevent long-running database operations
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel() // Ensure resources are released when function exits

	// Execute the query and scan the new version number into the user struct
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&user.Version)
	if err != nil {
		switch {
		// Handle case where email already exists in database (unique constraint violation)
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		// Handle case where version doesn't match (optimistic locking conflict)
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		// For all other errors, return them directly
		default:
			return err
		}
	}

	// Return nil if the update was successful
	return nil
}
