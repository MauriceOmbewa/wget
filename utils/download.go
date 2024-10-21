package utils

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

// Common browser User-Agents to rotate between for more natural behavior
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:122.0) Gecko/20100101 Firefox/122.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2.1 Safari/605.1.15",
}

// Accept headers for different content types, mimicking browser behavior
var acceptHeaders = map[string]string{
	"html":    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
	"image":   "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8",
	"js":      "text/javascript,application/javascript,application/ecmascript,application/x-ecmascript",
	"css":     "text/css,*/*;q=0.1",
	"default": "*/*",
}

// getRandomUserAgent returns a random User-Agent string from the predefined list
// This helps prevent detection of automated downloads
func getRandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// getAcceptHeader returns the appropriate Accept header based on the file type
// This mimics how browsers send different Accept headers for different resource types
func getAcceptHeader(fileName string) string {
	ext := strings.ToLower(getFileExtension(fileName))
	switch {
	case ext == "html" || ext == "htm":
		return acceptHeaders["html"]
	case ext == "jpg" || ext == "jpeg" || ext == "png" || ext == "gif" || ext == "webp" || ext == "svg":
		return acceptHeaders["image"]
	case ext == "js":
		return acceptHeaders["js"]
	case ext == "css":
		return acceptHeaders["css"]
	default:
		return acceptHeaders["default"]
	}
}

// getFileExtension extracts the file extension from a filename
// Returns an empty string if no extension is found
func getFileExtension(fileName string) string {
	parts := strings.Split(fileName, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}

// createBrowserLikeRequest creates an http.Request with headers that mimic a real browser
// This helps bypass restrictions that might be in place against automated downloads
//
// Parameters:
//   - urlStr: The URL to download from
//   - fileName: The name of the file being downloaded (used to set appropriate headers)
//
// Returns:
//   - *http.Request: The created request
//   - error: Any error that occurred during request creation
func createBrowserLikeRequest(urlStr string, fileName string) (*http.Request, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	// Set common browser headers
	req.Header.Set("User-Agent", getRandomUserAgent())
	req.Header.Set("Accept", getAcceptHeader(fileName))
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Cache-Control", "max-age=0")

	// Add Referer if URL has a path
	parsedURL := strings.Split(urlStr, "/")
	if len(parsedURL) > 3 {
		baseURL := strings.Join(parsedURL[:3], "/")
		req.Header.Set("Referer", baseURL)
	}

	return req, nil
}

// DownloadFile downloads a file from the specified URL while imitating browser behavior
//
// Parameters:
//   - urlStr: The URL to download from
//   - fileName: The name to save the file as
//   - background: If true, downloads in background mode without progress bar
//   - rateLimit: Download speed limit in bytes per second (0 for unlimited)
//
// Returns:
//   - error: Any error that occurred during the download process
//
// The function provides progress updates and timing information unless running
// in background mode. It also handles rate limiting if specified.
func DownloadFile(urlStr, fileName string, background bool, rateLimit int64) error {
	startTime := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("start at %s\n", startTime)

	// Create a custom client with appropriate timeout and redirect handling
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			// Copy headers to redirected request
			for key, values := range via[0].Header {
				req.Header[key] = values
			}
			return nil
		},
	}

	// Create and send browser-like request
	req, err := createBrowserLikeRequest(urlStr, fileName)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error: got status %s", resp.Status)
	}
	fmt.Printf("sending request, awaiting response... status %s\n", resp.Status)

	contentLength := resp.ContentLength
	fmt.Printf("content size: %d [~%.2fMB]\n", contentLength, float64(contentLength)/1000/1000)

	// Create output file
	out, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}
	defer out.Close()

	fmt.Printf("saving file to: ./%s\n", fileName)

	// Set up rate limiting if requested
	var reader io.Reader = resp.Body
	if rateLimit > 0 {
		fmt.Printf("Rate limit set to: %.2f KB/s\n", float64(rateLimit)/1024)
		reader = NewRateLimitReader(resp.Body, rateLimit)
	}

	// Download with or without progress bar based on background mode
	if background {
		_, err := io.Copy(out, reader)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}
	} else {
		bar := NewProgressBar(contentLength, 50)
		bar.StartTimer()

		_, err = io.Copy(io.MultiWriter(out, bar), reader)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}
	}

	endTime := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("Downloaded [%s]\nfinished at %s\n", urlStr, endTime)

	return nil
}

// DownloadWithLogging wraps DownloadFile with logging capabilities
//
// Parameters:
//   - urlStr: The URL to download from
//   - fileName: The name to save the file as
//   - background: If true, downloads in background mode and logs to 'wget-log'
//   - rateLimit: Download speed limit in bytes per second (0 for unlimited)
//
// If background is true, all output is written to 'wget-log' instead of stdout.
// The function returns immediately in background mode but continues downloading.
func DownloadWithLogging(urlStr string, fileName string, background bool, rateLimit int64) {
	// Initialize random seed for user agent selection
	rand.Seed(time.Now().UnixNano())

	if background {
		fmt.Println("Output will be written to 'wget-log'.")
		go func() {
			logFile, err := os.Create("wget-log")
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			defer logFile.Close()

			os.Stdout = logFile
			os.Stderr = logFile

			err1 := DownloadFile(urlStr, fileName, background, rateLimit)
			if err1 != nil {
				fmt.Fprintln(logFile, "Error:", err1)
			}
		}()
		time.Sleep(3 * time.Second)
	} else {
		err := DownloadFile(urlStr, fileName, background, rateLimit)
		if err != nil {
			fmt.Println(err)
		}
	}
}

// GetFileName extracts the filename from a URL
// Returns the last component of the URL path
func GetFileName(url string) string {
	s := strings.Split(url, "/")
	return s[len(s)-1]
}