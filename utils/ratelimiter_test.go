package utils

import (
	"bytes"
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

// TestNewRateLimitReader tests the NewRateLimitReader function
func TestNewRateLimitReader(t *testing.T) {
	reader := bytes.NewReader([]byte("Hello, World!"))
	rateLimit := int64(1024) // 1KB/s
	rlReader := NewRateLimitReader(reader, rateLimit)

	if rlReader.reader == nil {
		t.Error("NewRateLimitReader returned a nil reader")
	}
	if rlReader.rateLimit != rateLimit {
		t.Errorf("NewRateLimitReader set rateLimit = %d; want %d", rlReader.rateLimit, rateLimit)
	}
	if rlReader.bucketSize != rateLimit {
		t.Errorf("NewRateLimitReader set bucketSize = %d; want %d", rlReader.bucketSize, rateLimit)
	}
}
