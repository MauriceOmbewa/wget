package utils

import (
	"bytes"
	"testing"
	"time"
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

func TestProgressBar_EndTimer(t *testing.T) {
	pb := NewProgressBar(1000, 20)
	pb.StartTimer()

	// Simulate some download time
	time.Sleep(100 * time.Millisecond)

	duration := pb.EndTimer()
	if duration < 100*time.Millisecond {
		t.Errorf("expected duration to be at least 100ms, got %v", duration)
	}
}

func TestProgressBar_CalculateSpeed(t *testing.T) {
	total := int64(1000)
	barLength := 20
	pb := NewProgressBar(total, barLength)

	pb.StartTimer()
	time.Sleep(100 * time.Millisecond) // Simulate download time

	// Simulate writing data
	for i := 0; i < 10; i++ {
		pb.Write(bytes.Repeat([]byte("x"), 100)) // Simulate 1000 bytes total
	}

	speed := pb.CalculateSpeed()
	if speed <= 0 {
		t.Errorf("expected speed to be greater than 0, got %f", speed)
	}
}
