package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalidRuntimeFormat is an error variable that indicates when a runtime format is invalid.
// This error is returned when parsing or processing runtime data in an unexpected format.
var ErrInvalidRuntimeFormat = errors.New("invalid runtime format")

// Custom Runtime type
type Runtime int32

// MarshalJSON customizes the JSON marshaling for Runtime values.
// Converts the Runtime integer to a human-friendly string format with " mins" suffix,
// then safely quotes the string for proper JSON encoding.
// Example: 102 becomes "102 mins" as a JSON string.
func (r Runtime) MarshalJSON() ([]byte, error) {
	// Format the runtime as "X mins" string
	jsonValue := fmt.Sprintf("%d mins", r)

	// Quote the string to make it a valid JSON string value
	quotedJSONValue := strconv.Quote(jsonValue)

	// Convert to byte slice (never returns an error - satisfies json.Marshaler interface)
	return []byte(quotedJSONValue), nil
}

// UnmarshalJSON customizes the JSON unmarshaling for Runtime values.
// It expects a string in the format "X mins" and converts it to an integer.
// - Unquotes the JSON string to get the raw value.
// - Splits the string into parts based on spaces.
// - Validates that there are two parts and the second part is "mins".
// - Parses the first part as an integer.
// - Assigns the parsed integer to the Runtime receiver.
// Returns an error if the format is invalid or parsing fails.
func (r *Runtime) UnmarshalJSON(jsonValue []byte) error {
	// Remove the quotes from the JSON string.
	unquotedJSONValue, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		// If unquoting fails, the format is invalid.
		return ErrInvalidRuntimeFormat
	}

	// Split the string into parts based on spaces.
	parts := strings.Split(unquotedJSONValue, " ")

	// Check if there are exactly two parts and the second part is "mins".
	if len(parts) != 2 || parts[1] != "mins" {
		// If not, the format is invalid.
		return ErrInvalidRuntimeFormat
	}

	// Parse the first part as an integer.
	i, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		// If parsing fails, the format is invalid.
		return ErrInvalidRuntimeFormat
	}

	// Assign the parsed integer to the Runtime receiver.
	*r = Runtime(i)

	return nil
}
