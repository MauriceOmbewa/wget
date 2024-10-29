package utils

import (
	"bytes"
	"io"
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

// TestRateLimitReaderRead tests the Read method of RateLimitReader
func TestRateLimitReaderRead(t *testing.T) {
	t.Run("Read less than limit", func(t *testing.T) {
		originalData := []byte("Hello, World!")
		reader := bytes.NewReader(originalData)
		rateLimit := int64(1024) // 1KB/s
		rlReader := NewRateLimitReader(reader, rateLimit)

		buf := make([]byte, 5)
		n, err := rlReader.Read(buf)
		if err != nil {
			t.Fatalf("Read returned an error: %v", err)
		}
		if n != 5 {
			t.Errorf("Read = %d; want 5", n)
		}
		if string(buf) != "Hello" {
			t.Errorf("Read data = %q; want %q", buf, "Hello")
		}
	})

	t.Run("Read exceeds bucket", func(t *testing.T) {
		originalData := []byte("Hello, World!")
		reader := bytes.NewReader(originalData)
		rateLimit := int64(5) // 5B/s
		rlReader := NewRateLimitReader(reader, rateLimit)

		buf := make([]byte, 20)
		// Read more than the available bytes in the bucket
		n, err := rlReader.Read(buf)

		// We expect to read only a part of the data due to the rate limiting
		if err != nil && err != io.EOF {
			t.Fatalf("Read returned an error: %v", err)
		}

		if n > 5 {
			t.Errorf("Read = %d; want <= 5", n)
		}
	})

	t.Run("Rate limit empty bucket", func(t *testing.T) {
		// Test when the rate limit is set to zero
		originalData := []byte("Hello, World!")
		reader := bytes.NewReader(originalData)
		rateLimit := int64(0) // No limit
		rlReader := NewRateLimitReader(reader, rateLimit)

		buf := make([]byte, 20)
		n, err := rlReader.Read(buf)
		if err != nil {
			t.Fatalf("Read returned an error: %v", err)
		}
		if n != len(originalData) {
			t.Errorf("Read = %d; want %d", n, len(originalData))
		}
		if string(buf[:n]) != string(originalData) {
			t.Errorf("Read data = %q; want %q", buf[:n], originalData)
		}
	})
}
