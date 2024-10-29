package utils

import (
	"bytes"
	"testing"
)

// Mocked stdout for capturing print statements
var stdout = &bytes.Buffer{}

func TestProgressBar_Write(t *testing.T) {
	total := int64(1000) // Total bytes to download
	barLength := 20      // Length of the progress bar
	pb := NewProgressBar(total, barLength)

	// Start the timer
	pb.StartTimer()

	// Simulate writing some data
	data := []byte("hello world")
	n, err := pb.Write(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n != len(data) {
		t.Errorf("expected %d bytes written, got %d", len(data), n)
	}

	// Check if the progress bar is updated correctly
	if pb.Written != int64(len(data)) {
		t.Errorf("expected %d bytes written, got %d", len(data), pb.Written)
	}
}
