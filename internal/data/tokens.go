package data

import (
	"crypto/rand"
	"crypto/sha256"
	"time"
)

// constants to represent activation
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
