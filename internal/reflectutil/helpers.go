package reflectutil

import (
	"reflect"
	"strconv"
)

// IsTrue parses a string as a boolean value.
// Returns true if the string represents a true value, false otherwise.
// Ignores parsing errors and returns false as the default.
func IsTrue(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

// IsIntegerKind checks if the given reflect.Kind represents an integer type.
// Returns true for all signed and unsigned integer types (Int, Int8, Int16,
// Int32, Int64, Uint, Uint8, Uint16, Uint32, Uint64).
// Returns false for Uintptr and all other types.
func IsIntegerKind(k reflect.Kind) bool {
	return k >= reflect.Int &&
		k <= reflect.Uint64 &&
		k != reflect.Uintptr
}
