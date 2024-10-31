package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
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

	// Extract and download resources
	htmlContent := string(body)
	resourceMap := DownloadResources(htmlContent, pageURL, baseFolder)

	// Update HTML content with local paths
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

	// Save the modified HTML content to a file
	htmlPath := filepath.Join(baseFolder, "index.html")
	err = os.WriteFile(htmlPath, []byte(htmlContent), 0o644)
	if err != nil {
		return err
	}

	return nil
}

// DownloadResources fetches images, CSS, and JavaScript files using regex to find URLs
func DownloadResources(htmlContent, pageURL, baseFolder string) map[string]string {
	resourceMap := make(map[string]string)
	var mutex sync.Mutex
	var wg sync.WaitGroup
	
	// Create a channel to limit concurrent downloads
	semaphore := make(chan struct{}, 5)

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`src="([^"]*?)"`),
		regexp.MustCompile(`href="([^"]*?)"`),
		regexp.MustCompile(`url\(['"]?([^'"()]+)['"]?\)`),
		regexp.MustCompile(`@import\s+['"]([^'"]+)['"]`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(htmlContent, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			resourceURL := match[1]

			// Skip if it's a data URL, anchor, or javascript
			if strings.HasPrefix(resourceURL, "data:") ||
				strings.HasPrefix(resourceURL, "#") ||
				strings.HasPrefix(resourceURL, "javascript:") {
				continue
			}

			// Resolve the absolute URL
			absoluteURL := resolveURL(pageURL, resourceURL)
			if absoluteURL == "" {
				fmt.Printf("Failed to resolve URL: %s\n", resourceURL)
				continue
			}

			// Skip if resource is from a different domain
			if !isSameDomain(pageURL, absoluteURL) {
				fmt.Printf("Skipping external resource: %s\n", absoluteURL)
				continue
			}

			wg.Add(1)
			go func(absURL, resURL string) {
				defer wg.Done()
				
				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// Download the file
				if filename, err := DownloadFile(absURL, baseFolder); err == nil {
					mutex.Lock()
					resourceMap[resURL] = filename
					mutex.Unlock()
					fmt.Printf("Downloaded: %s -> %s\n", absURL, filename)
				} else {
					fmt.Printf("Failed to download %s: %v\n", absURL, err)
				}
			}(absoluteURL, resourceURL)
		}
	}

	wg.Wait()
	return resourceMap
}

// DownloadFile downloads a file and saves it locally
func DownloadFile(fileURL, baseFolder string) (string, error) {
	if !shouldDownloadFile(fileURL) {
		return "", fmt.Errorf("file filtered out: %s", fileURL)
	}

	resp, err := http.Get(fileURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Extract filename from URL
	u, err := url.Parse(fileURL)
	if err != nil {
		return "", err
	}
	filename := path.Base(u.Path)

	// Save file to local directory
	if filename != "" {
		filePath := filepath.Join(baseFolder, filename)
		out, err := os.Create(filePath)
		if err != nil {
			return "", err
		}
		defer out.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return "", err
		}
		fmt.Printf("Downloaded: %s\n", filename)
		return filename, nil
	}
	return "", fmt.Errorf("empty filename")
}

// Resolve relative URLs to absolute
func resolveURL(baseURL, resourcePath string) string {
	// If it's already an absolute URL, return it
	if strings.HasPrefix(resourcePath, "http://") || strings.HasPrefix(resourcePath, "https://") {
		return resourcePath
	}

	// Parse the base URL
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}

	// Handle absolute paths (starting with /)
	if strings.HasPrefix(resourcePath, "/") {
		base.Path = resourcePath
		return base.String()
	}

	// Handle relative paths
	rel, err := url.Parse(resourcePath)
	if err != nil {
		return ""
	}

	return base.ResolveReference(rel).String()
}

// Add these new types and variables at the top of the file
type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

var (
	excludePatterns stringSliceFlag
	rejectPatterns  stringSliceFlag
)

// Add this new function
func shouldDownloadFile(fileURL string) bool {
	filename := path.Base(fileURL)
	extension := strings.ToLower(path.Ext(filename))

	// Check reject patterns first
	for _, pattern := range rejectPatterns {
		if strings.Contains(extension, pattern) || strings.Contains(filename, pattern) {
			fmt.Printf("Rejected: %s (matched pattern: %s)\n", fileURL, pattern)
			return false
		}
	}

	// If exclude patterns exist, file must match at least one to be downloaded
	if len(excludePatterns) > 0 {
		for _, pattern := range excludePatterns {
			if strings.Contains(extension, pattern) || strings.Contains(filename, pattern) {
				return true
			}
		}
		fmt.Printf("Excluded: %s (didn't match any include patterns)\n", fileURL)
		return false
	}

	return true
}

// Add this function near the top of the file
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

func main() {
	// Define flags
	flag.Var(&excludePatterns, "X", "File pattern to exclude (can be used multiple times)")
	flag.Var(&rejectPatterns, "R", "File pattern to reject (can be used multiple times)")

	// Parse flags
	flag.Parse()

	// Get the baseURL from remaining arguments
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: mirror [flags] URL")
		flag.PrintDefaults()
		return
	}
	baseURL := args[0]

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
