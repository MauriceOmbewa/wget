package utils

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
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

func TestDownloadFilesConcurrently(t *testing.T) {
	// Set up a test server that serves a sample response for download
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	// Mock URLs pointing to the test server
	urls := []string{server.URL, server.URL}

	// Create a temporary directory to store downloaded files
	outputDir, err := os.MkdirTemp("", "downloads")
	if err != nil {
		t.Fatalf("could not create temp dir: %v", err)
	}
	defer os.RemoveAll(outputDir)

	// Test DownloadFilesConcurrently function
	outputPrefix := "test_file"
	rateLimit := int64(1024) // Set to 1KB/s for testing rate limiting

	err = DownloadFilesConcurrently(urls, outputPrefix, false, rateLimit, outputDir)
	if err != nil {
		t.Fatalf("DownloadFilesConcurrently failed: %v", err)
	}

	// Verify that files were downloaded
	for i := range urls {
		filename := filepath.Join(outputDir, outputPrefix+"_"+strconv.Itoa(i))
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			t.Errorf("expected file %s to be downloaded, but it was not found", filename)
		}
	}
}
