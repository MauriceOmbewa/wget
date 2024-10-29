package utils

import (
	"os"
	"testing"
)

func TestReadUrlsFromFile(t *testing.T) {
	// Create a temporary file with mock URLs
	tmpFile, err := os.CreateTemp("", "urls.txt")
	if err != nil {
		t.Fatalf("could not create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test URLs to file
	content := "http://example.com/file1\nhttp://example.com/file2"
	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatalf("could not write to temp file: %v", err)
	}

	// Test ReadUrlsFromFile function
	urls, err := ReadUrlsFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("ReadUrlsFromFile failed: %v", err)
	}

	// Expected output
	expected := []string{"http://example.com/file1", "http://example.com/file2"}
	if len(urls) != len(expected) {
		t.Errorf("expected %d URLs, got %d", len(expected), len(urls))
	}
	for i, url := range urls {
		if url != expected[i] {
			t.Errorf("expected URL %s, got %s", expected[i], url)
		}
	}
}
