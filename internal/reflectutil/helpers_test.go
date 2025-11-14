package reflectutil

import "testing"

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
