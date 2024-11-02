package utils

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// MirrorWebsite initiates the website mirroring process. It creates a base directory
// named after the website's domain and starts downloading the website content.
func MirrorWebsite(baseURL string, reject []string, exclude []string, convertLinks bool) error {
	fmt.Printf("\n=== Starting mirror of %s ===\n", baseURL)
	baseFolder, err := createDirectory(baseURL)
	if err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}
	fmt.Printf("Created directory: %s\n\n", baseFolder)
	return downloadPage(baseURL, baseFolder, reject, exclude, convertLinks)
}

// createDirectory creates a directory named after the website's domain.
// It returns the created directory path or an error if creation fails.
func createDirectory(baseURL string) (string, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	domain := parsedURL.Host
	err = os.MkdirAll(domain, 0755)
	return domain, err
}

// downloadPage downloads a single webpage and its resources. It processes the HTML content,
// downloads all associated resources, and updates links in the HTML if specified.
// The page is saved maintaining the original URL path structure.
func downloadPage(pageURL, baseFolder string, reject []string, exclude []string, convertLinks bool) error {
	fmt.Printf("Downloading page: %s\n", pageURL)
	resp, err := http.Get(pageURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch page: %s", pageURL)
	}
	fmt.Printf("Got response: %s for %s\n", resp.Status, pageURL)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	htmlContent := string(body)
	resourceMap := downloadResources(htmlContent, pageURL, baseFolder, reject, exclude)

	if convertLinks {
		htmlContent = updateLinks(htmlContent, resourceMap)
	} else {
		htmlContent = updateCSSJSPaths(htmlContent, resourceMap)
	}

	// Get the URL path
	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		return err
	}
	// Determine the path for saving the HTML file
	relativePath := strings.TrimPrefix(parsedURL.Path, "/")
	fmt.Println("Relative path:", relativePath)
	if relativePath == "" {
		relativePath = "index.html"
	} else if !strings.Contains(relativePath, ".") {
		fmt.Println("No extension found, checking if content is HTML")
		// Create a temporary file to check if content is HTML
		tempPath := filepath.Join(baseFolder, relativePath)
		if err := os.MkdirAll(tempPath, 0755); err != nil {
			return fmt.Errorf("failed to create temp directory: %v", err)
		}
		
		tempFile := filepath.Join(tempPath, "temp")
		if err := os.WriteFile(tempFile, []byte(htmlContent), 0644); err != nil {
			return fmt.Errorf("failed to write temp file: %v", err)
		}
		
		// Check if the content is HTML
		if isHTMLFile(tempFile) {
			relativePath = filepath.Join(relativePath, "index.html")
			fmt.Printf("Detected HTML content, using path: %s\n", relativePath)
		}
		
		// Clean up temp file
		os.Remove(tempFile)
	}

	// Create all necessary directories
	dir := filepath.Join(baseFolder, filepath.Dir(relativePath))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directories: %v", err)
	}

	// Save the HTML file
	htmlPath := filepath.Join(baseFolder, relativePath)
	fmt.Printf("Saving HTML to: %s\n", filepath.Join(baseFolder, relativePath))
	return os.WriteFile(htmlPath, []byte(htmlContent), 0644)
}

// updateCSSJSPaths modifies CSS and JavaScript file paths in HTML content to use relative paths.
// It updates both src and href attributes to point to the locally downloaded files.
func updateCSSJSPaths(htmlContent string, resourceMap map[string]string) string {
	for originalURL, filename := range resourceMap {
		if strings.HasSuffix(strings.ToLower(originalURL), ".css") || strings.HasSuffix(strings.ToLower(originalURL), ".js") {
			// Handle src attribute with both quote types
			htmlContent = strings.ReplaceAll(htmlContent,
				fmt.Sprintf(`src="%s"`, originalURL),
				fmt.Sprintf(`src="./%s"`, filename))
			htmlContent = strings.ReplaceAll(htmlContent,
				fmt.Sprintf(`src='%s'`, originalURL),
				fmt.Sprintf(`src='./%s'`, filename))
			
			// Handle href attribute for CSS files with both quote types
			htmlContent = strings.ReplaceAll(htmlContent,
				fmt.Sprintf(`href="%s"`, originalURL),
				fmt.Sprintf(`href="./%s"`, filename))
			htmlContent = strings.ReplaceAll(htmlContent,
				fmt.Sprintf(`href='%s'`, originalURL),
				fmt.Sprintf(`href='./%s'`, filename))
		}
	}
	return htmlContent
}
//bool to check whether to add .html to files that don't have an extension and are html files if we look at the content
func isHTMLFile(path string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), "<html") || strings.Contains(string(content), "<!DOCTYPE html")
}


// updateLinks modifies all resource links in HTML content to use relative paths.
// This includes images, stylesheets, scripts, and other embedded resources.
func updateLinks(htmlContent string, resourceMap map[string]string) string {
	for originalURL, filename := range resourceMap {
		htmlContent = strings.ReplaceAll(htmlContent,
			fmt.Sprintf(`src="%s"`, originalURL),
			fmt.Sprintf(`src="./%s"`, filename))
		htmlContent = strings.ReplaceAll(htmlContent,
			fmt.Sprintf(`href="%s"`, originalURL),
			fmt.Sprintf(`href="./%s"`, filename))
		htmlContent = strings.ReplaceAll(htmlContent,
			fmt.Sprintf(`url(%s)`, originalURL),
			fmt.Sprintf(`url("./%s")`, filename))
		htmlContent = strings.ReplaceAll(htmlContent,
			fmt.Sprintf(`url('%s')`, originalURL),
			fmt.Sprintf(`url('./%s')`, filename))
		htmlContent = strings.ReplaceAll(htmlContent,
			fmt.Sprintf(`url("%s")`, originalURL),
			fmt.Sprintf(`url("./%s")`, filename))
	}
	return htmlContent
}

// downloadResources scans HTML content for resources (images, scripts, stylesheets, etc.)
// and downloads them concurrently. It maintains a map of original URLs to local file paths.
// Uses a semaphore to limit concurrent downloads.
func downloadResources(htmlContent, pageURL, baseFolder string, reject []string, exclude []string) map[string]string {
	fmt.Printf("\nScanning for resources in: %s\n", pageURL)
	resourceMap := make(map[string]string)
	var mutex sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, 5)

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`src=['"]([^'"]*?)['"]`),                              // src with both quote types
		regexp.MustCompile(`href=['"]([^'"]*?)['"]`),                             // href with both quote types
		regexp.MustCompile(`url\(['"]?([^'"()]+)['"]?\)`),                        // CSS url()
		regexp.MustCompile(`@import\s+['"]([^'"]+)['"]`),                         // CSS @import
		regexp.MustCompile(`<script[^>]+src=['"]([^'"]+)['"]`),                   // script tags
		regexp.MustCompile(`<link[^>]+href=['"]([^'"]+)['"]`),                    // link tags
		regexp.MustCompile(`<img[^>]+src=['"]([^'"]+)['"]`),                      // img tags
		regexp.MustCompile(`content=['"]([^'"]+\.(?:png|jpg|jpeg|gif|ico))['"]`), // meta images
	}

	processedURLs := make(map[string]bool)
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(htmlContent, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			resourceURL := match[1]

			if processedURLs[resourceURL] {
				continue
			}
			processedURLs[resourceURL] = true

			if shouldSkipResource(resourceURL) {
				continue
			}

			absoluteURL := resolveURL(pageURL, resourceURL)
			if absoluteURL == "" || !isSameDomain(pageURL, absoluteURL) {
				continue
			}

			wg.Add(1)
			go func(absURL, resURL string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				if filename, err := downloadFile(absURL, baseFolder, reject, exclude); err == nil {
					mutex.Lock()
					resourceMap[resURL] = filename
					mutex.Unlock()

					if strings.HasSuffix(strings.ToLower(filename), ".css") {
						if cssContent, err := os.ReadFile(filepath.Join(baseFolder, filename)); err == nil {
							cssResources := downloadCSSResources(string(cssContent), absURL, baseFolder, reject, exclude)
							mutex.Lock()
							for k, v := range cssResources {
								resourceMap[k] = v
							}
							mutex.Unlock()
						}
					}
				}
			}(absoluteURL, resourceURL)
		}
	}

	wg.Wait()
	return resourceMap
}

// downloadCSSResources scans CSS content for referenced resources (like images and fonts)
// and downloads them concurrently. Similar to downloadResources but specific to CSS files.
func downloadCSSResources(cssContent, baseURL, baseFolder string, reject []string, exclude []string) map[string]string {
	fmt.Printf("Scanning CSS for resources from: %s\n", baseURL)
	resourceMap := make(map[string]string)
	var mutex sync.Mutex
	var wg sync.WaitGroup

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`url\(['"]?([^'"()]+)['"]?\)`),
		regexp.MustCompile(`@import\s+['"]([^'"]+)['"]`),
	}

	semaphore := make(chan struct{}, 5)

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(cssContent, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			resourceURL := match[1]

			if shouldSkipResource(resourceURL) {
				continue
			}

			absoluteURL := resolveURL(baseURL, resourceURL)
			if absoluteURL == "" || !isSameDomain(baseURL, absoluteURL) {
				continue
			}

			wg.Add(1)
			go func(absURL, resURL string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				if filename, err := downloadFile(absURL, baseFolder, reject, exclude); err == nil {
					mutex.Lock()
					resourceMap[resURL] = filename
					mutex.Unlock()
				}
			}(absoluteURL, resourceURL)
		}
	}

	wg.Wait()
	return resourceMap
}

// shouldSkipResource checks if a resource URL should be skipped based on its scheme
// or if it's a special URL type (like data: URLs, javascript:, mailto:, etc.)
func shouldSkipResource(resourceURL string) bool {
	return strings.HasPrefix(resourceURL, "data:") ||
		strings.HasPrefix(resourceURL, "#") ||
		strings.HasPrefix(resourceURL, "javascript:") ||
		strings.HasPrefix(resourceURL, "mailto:") ||
		strings.HasPrefix(resourceURL, "tel:") ||
		resourceURL == "" ||
		resourceURL == "/"
}

// downloadFile downloads a single file from fileURL and saves it to the appropriate
// location in baseFolder, maintaining the original path structure.
// Returns the relative path to the downloaded file or an error.
func downloadFile(fileURL, baseFolder string, reject []string, exclude []string) (string, error) {
	if !shouldDownloadFile(fileURL, reject, exclude) {
		fmt.Printf("Skipping filtered file: %s\n", fileURL)
		return "", fmt.Errorf("file filtered out: %s", fileURL)
	}

	fmt.Printf("Downloading resource: %s\n", fileURL)
	resp, err := http.Get(fileURL)
	if err != nil {
		fmt.Printf("Error downloading %s: %v\n", fileURL, err)
		return "", err
	}
	defer resp.Body.Close()

	u, err := url.Parse(fileURL)
	if err != nil {
		return "", err
	}

	// Get the path without leading slash
	relativePath := strings.TrimPrefix(u.Path, "/")
	if relativePath == "" {
		return "", fmt.Errorf("empty path")
	}

	// Create the full path including folders
	fullPath := filepath.Join(baseFolder, relativePath)

	// Create all necessary directories
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directories: %v", err)
	}

	// Create and write to the file
	out, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	// After successful download
	fmt.Printf("Successfully downloaded: %s -> %s/%s\n", fileURL, baseFolder, relativePath)
	return relativePath, nil
}

// shouldDownloadFile checks if a file should be downloaded based on reject and exclude patterns.
// Returns false if the file matches any reject pattern or exclude pattern.
func shouldDownloadFile(fileURL string, reject []string, exclude []string) bool {
	u, err := url.Parse(fileURL)
	if err != nil {
		return false
	}

	urlPath := u.Path

	// First check rejects - if any match, don't download
	for _, pattern := range reject {
		if strings.Contains(urlPath, pattern) {
			return false
		}
	}

	// If exclude patterns exist, return false only if the path matches an exclude pattern
	if len(exclude) > 0 {
		for _, pattern := range exclude {
			if strings.HasPrefix(urlPath, pattern) {
				return false // Skip this file as it matches exclude pattern
			}
		}
	}

	// If we get here, the file should be downloaded
	return true
}

// resolveURL converts a relative URL to an absolute URL using the base URL.
// Handles both absolute URLs and relative URLs (with or without leading slash).
func resolveURL(baseURL, resourcePath string) string {
	if strings.HasPrefix(resourcePath, "http://") || strings.HasPrefix(resourcePath, "https://") {
		return resourcePath
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}

	if strings.HasPrefix(resourcePath, "/") {
		base.Path = resourcePath
		return base.String()
	}

	rel, err := url.Parse(resourcePath)
	if err != nil {
		return ""
	}

	return base.ResolveReference(rel).String()
}

// isSameDomain checks if two URLs belong to the same domain.
// Used to ensure we only download resources from the target website.
func isSameDomain(baseURL, resourceURL string) bool {
	base, err := url.Parse(baseURL)
	if err != nil {
		return false
	}

	resource, err := url.Parse(resourceURL)
	if err != nil {
		return false
	}

	return base.Host == resource.Host
}
