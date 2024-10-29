package utils

import (
	"testing"
)

// TestParseRateLimit tests the ParseRateLimit function
func TestParseRateLimit(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		err      bool
	}{
		{"400k", 400 * 1024, false},
		{"2M", 2 * 1024 * 1024, false},
		{"", 0, false},
		{"invalid", 0, true},
		{"1.5M", 0, true}, // Expecting it to fail because of the decimal
	}

	for _, test := range tests {
		result, err := ParseRateLimit(test.input)
		if (err != nil) != test.err {
			t.Errorf("ParseRateLimit(%q) returned error: %v, expected error: %v", test.input, err, test.err)
		}
		if result != test.expected {
			t.Errorf("ParseRateLimit(%q) = %d; want %d", test.input, result, test.expected)
		}
	}
}
