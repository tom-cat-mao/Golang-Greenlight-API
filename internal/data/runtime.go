package data

import (
	"fmt"
	"strconv"
)

// Custom Runtime type
type Runtime int32

// MarshJSON customizes the JSON marshaling for Runtime values.
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
