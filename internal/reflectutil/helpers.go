package reflectutil

import "strconv"

// IsTrue parses a string as a boolean value.
// Returns true if the string represents a true value, false otherwise.
// Ignores parsing errors and returns false as the default.
func IsTrue(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}
