package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// CreateDirectory creates a directory for saving the mirrored site
func CreateDirectory(baseURL string) (string, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	domain := parsedURL.Host
	err = os.MkdirAll(domain, 0o755)
	return domain, err
}

// DownloadPage downloads the HTML page and its assets
func DownloadPage(pageURL, baseFolder string) error {
	resp, err := http.Get(pageURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch page: %s", pageURL)
	}

	// Read the entire HTML content
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Save the HTML content to a file
	htmlPath := filepath.Join(baseFolder, "index.html")
	err = os.WriteFile(htmlPath, body, 0o644)
	if err != nil {
		return err
	}

	// Extract and download resources
	htmlContent := string(body)
	DownloadResources(htmlContent, pageURL, baseFolder)

	return nil
}

// DownloadResources fetches images, CSS, and JavaScript files using regex to find URLs
func DownloadResources(htmlContent, pageURL, baseFolder string) {
	// Regex patterns for finding src and href attributes
	imgRegex := regexp.MustCompile(`src="(.*?)"`)
	cssJsRegex := regexp.MustCompile(`href="(.*?)"`)

	// Find and download all resources
	for _, match := range imgRegex.FindAllStringSubmatch(htmlContent, -1) {
		resourceURL := match[1]
		absoluteURL := resolveURL(pageURL, resourceURL)
		DownloadFile(absoluteURL, baseFolder)
	}
	for _, match := range cssJsRegex.FindAllStringSubmatch(htmlContent, -1) {
		resourceURL := match[1]
		absoluteURL := resolveURL(pageURL, resourceURL)
		DownloadFile(absoluteURL, baseFolder)
	}
}

// DownloadFile downloads a file and saves it locally
func DownloadFile(fileURL, baseFolder string) error {
	resp, err := http.Get(fileURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Extract filename from URL
	u, err := url.Parse(fileURL)
	if err != nil {
		return err
	}
	filename := path.Base(u.Path)

	// Save file to local directory
	if filename != "" {
		filePath := filepath.Join(baseFolder, filename)
		out, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return err
		}
		fmt.Printf("Downloaded: %s\n", filename)
	}
	return nil
}

// Resolve relative URLs to absolute
func resolveURL(baseURL, resourceURL string) string {
	if strings.HasPrefix(resourceURL, "http") {
		return resourceURL
	}
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	absoluteURL := parsedBaseURL.Scheme + "://" + parsedBaseURL.Host + path.Join(path.Dir(parsedBaseURL.Path), resourceURL)
	return absoluteURL
}

func main() {
	baseURL := os.Args[1]
	baseFolder, err := CreateDirectory(baseURL)
	if err != nil {
		fmt.Println("Error creating directory:", err)
		return
	}

	err = DownloadPage(baseURL, baseFolder)
	if err != nil {
		fmt.Println("Error downloading page:", err)
	}
}