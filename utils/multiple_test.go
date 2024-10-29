package utils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
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

func TestDownloadFileErrorHandling(t *testing.T) {
	// Use an invalid URL to trigger an error
	url := "http://invalid-url.com/file"
	fileName := "invalid_download_test.txt"

	err := DownloadFile(url, fileName, false, 0)
	if err == nil {
		t.Errorf("expected an error for an invalid URL, but got none")
	}

	// Check for common network-related errors
	if err != nil && !strings.Contains(err.Error(), "no such host") &&
		!strings.Contains(err.Error(), "dial tcp") &&
		!strings.Contains(err.Error(), "http") {
		t.Errorf("unexpected error for invalid URL: %v", err)
	}
}

func TestConcurrentDownloadWithRateLimit(t *testing.T) {
	// Set a low rate limit to test rate limiting functionality
	rateLimit := int64(512) // 512 bytes per second

	// Mock server to simulate content download
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate slow response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("rate-limited content"))
	}))
	defer server.Close()

	// Set up URLs for testing concurrent downloading
	urls := []string{server.URL, server.URL}

	// Create a temporary directory for downloaded files
	outputDir, err := os.MkdirTemp("", "rate_limited_downloads")
	if err != nil {
		t.Fatalf("could not create temp dir: %v", err)
	}
	defer os.RemoveAll(outputDir)

	// Test the DownloadFilesConcurrently function with rate limit
	err = DownloadFilesConcurrently(urls, "rate_test_file", false, rateLimit, outputDir)
	if err != nil {
		t.Fatalf("DownloadFilesConcurrently failed under rate limiting: %v", err)
	}

	// Verify downloaded files exist and contain expected data
	for i := range urls {
		filename := filepath.Join(outputDir, fmt.Sprintf("rate_test_file_%d", i))
		info, err := os.Stat(filename)
		if os.IsNotExist(err) {
			t.Errorf("file %s should exist but was not found", filename)
			continue
		}
		if err != nil {
			t.Errorf("could not stat file %s: %v", filename, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("file %s should have content but is empty", filename)
		}
	}
}
