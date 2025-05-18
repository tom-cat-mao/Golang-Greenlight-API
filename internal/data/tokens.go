package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"time"

	"greenlight.tomcat.net/internal/validator"
)

// Define constants for different token scopes.
// Activation scope is defined, used for user account activation.
// This constant helps categorize tokens and manage their purpose within the application.
const (
	ScopActivation = "activation"
)

// The Token struct
// hold the data for an individual token
// plaintext
// hashed versions of the token
// associated user ID
// expiry time
// scope
type Token struct {
	Plaintext string
	Hash      []byte
	UserID    int64
	Expiry    time.Time
	Scope     string
}

// TokenModel struct to include the sql connection
type TokenModel struct {
	DB *sql.DB
}

// Generate token for user activation
// parameter: userID
//
//	ttl time to live duration
//	scope scope of the token
func generateToken(userID int64, ttl time.Duration, scope string) *Token {
	token := &Token{
		Plaintext: rand.Text(), // set the Plaintext field to be a random token
		UserID:    userID,
		Expiry:    time.Now().Add(ttl),
		Scope:     scope,
	}

	// Generate a SHA-256 hash of the plaintext token string. This will be the value
	// that we store in the `hash` field of our database table. Note that the
	// sha256.Sum256() function returns an *array* of length 32, so to make it easier to
	// work with we convert it to a slice using the [:] operator before storing it
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token
}

// Validation check that the plaintext token has been
// provided and is exactly 26 bytes long
func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

// Shortcut which creates a new Token struct and then inserts
// the data in the tokens table
func (m TokenModel) New(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token := generateToken(userID, ttl, scope)

	err := m.Insert(token)
	return token, err
}

// Add the data for a specific token to the table
func (m TokenModel) Insert(token *Token) error {
	query := `
		INSERT INTO tokens (hash, user_id, expiry, scope)
		VALUES ($1, $2, $3, $4)
		`

	args := []any{token.Hash, token.UserID, token.Expiry, token.Scope}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, args...)
	return err

}

// Deletes all tokens for a specific user and scope
func (m TokenModel) DeleteAllForUser(scope string, userID int64) error {
	query := `
		DELETE FROM tokens
		WHERE scope = $1 AND user_id = $2
		`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, scope, userID)
	return err
}
