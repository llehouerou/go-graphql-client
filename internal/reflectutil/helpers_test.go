package reflectutil

import (
	"reflect"
	"testing"
)

func TestIsTrue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// True values
		{"true lowercase", "true", true},
		{"TRUE uppercase", "TRUE", true},
		{"True mixed case", "True", true},
		{"t lowercase", "t", true},
		{"T uppercase", "T", true},
		{"1 as string", "1", true},

		// False values
		{"false lowercase", "false", false},
		{"FALSE uppercase", "FALSE", false},
		{"False mixed case", "False", false},
		{"f lowercase", "f", false},
		{"F uppercase", "F", false},
		{"0 as string", "0", false},

		// Edge cases (invalid values default to false)
		{"empty string", "", false},
		{"invalid value", "invalid", false},
		{"yes", "yes", false},
		{"no", "no", false},
		{"whitespace", " ", false},
		{"number 2", "2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTrue(tt.input)
			if result != tt.expected {
				t.Errorf(
					"IsTrue(%q) = %v, expected %v",
					tt.input,
					result,
					tt.expected,
				)
			}
		})
	}
}

func TestIsIntegerKind(t *testing.T) {
	tests := []struct {
		name     string
		kind     reflect.Kind
		expected bool
	}{
		// Integer types (should return true)
		{"Int", reflect.Int, true},
		{"Int8", reflect.Int8, true},
		{"Int16", reflect.Int16, true},
		{"Int32", reflect.Int32, true},
		{"Int64", reflect.Int64, true},
		{"Uint", reflect.Uint, true},
		{"Uint8", reflect.Uint8, true},
		{"Uint16", reflect.Uint16, true},
		{"Uint32", reflect.Uint32, true},
		{"Uint64", reflect.Uint64, true},

		// Non-integer types (should return false)
		{"Uintptr", reflect.Uintptr, false},
		{"Float32", reflect.Float32, false},
		{"Float64", reflect.Float64, false},
		{"Bool", reflect.Bool, false},
		{"String", reflect.String, false},
		{"Array", reflect.Array, false},
		{"Slice", reflect.Slice, false},
		{"Struct", reflect.Struct, false},
		{"Ptr", reflect.Ptr, false},
		{"Interface", reflect.Interface, false},
		{"Map", reflect.Map, false},
		{"Chan", reflect.Chan, false},
		{"Func", reflect.Func, false},
		{"Invalid", reflect.Invalid, false},
		{"Complex64", reflect.Complex64, false},
		{"Complex128", reflect.Complex128, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsIntegerKind(tt.kind)
			if result != tt.expected {
				t.Errorf(
					"IsIntegerKind(%v) = %v, expected %v",
					tt.kind,
					result,
					tt.expected,
				)
			}
		})
	}
}
