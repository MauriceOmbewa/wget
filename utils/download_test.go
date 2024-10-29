package utils

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "download_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up

	// Create a test server that returns a fixed response
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Length", "1024")
		fmt.Fprint(w, "Test file content")
	}))
	defer ts.Close()

	tests := []struct {
		name        string
		url         string
		fileName    string
		background  bool
		rateLimit   int64
		expectError bool
	}{
		{
			name:        "Successful Download",
			url:         ts.URL,
			fileName:    filepath.Join(tempDir, "testfile.txt"),
			background:  false,
			rateLimit:   0,
			expectError: false,
		},
		{
			name:        "Download with Rate Limit",
			url:         ts.URL,
			fileName:    filepath.Join(tempDir, "testfile_rate_limited.txt"),
			background:  false,
			rateLimit:   100, // Limit to 100 bytes/sec
			expectError: false,
		},
		{
			name:        "Invalid URL",
			url:         "http://invalid-url",
			fileName:    filepath.Join(tempDir, "testfile_invalid.txt"),
			background:  false,
			rateLimit:   0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DownloadFile(tt.url, tt.fileName, tt.background, tt.rateLimit)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
			}

			// Check if the file was created on successful download
			if !tt.expectError {
				if _, err := os.Stat(tt.fileName); os.IsNotExist(err) {
					t.Errorf("file was not created: %s", tt.fileName)
				} else {
					// Clean up the file after test
					os.Remove(tt.fileName)
				}
			}
		})
	}
}

func TestGetFileName(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"http://example.com/file.txt", "file.txt"},
		{"http://example.com/path/to/file.txt", "file.txt"},
		{"http://example.com/", ""},
		{"http://example.com/path/to/", ""},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := GetFileName(tt.url)
			if result != tt.expected {
				t.Errorf("GetFileName(%q) = %q; want %q", tt.url, result, tt.expected)
			}
		})
	}
}
