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
)

func MirrorWebsite(baseURL string, reject []string, exclude []string, convertLinks bool) error {
	baseFolder, err := createDirectory(baseURL)
	if err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	visited := make(map[string]bool)
	err = downloadPage(baseURL, baseFolder, reject, exclude, convertLinks, visited)
	if err != nil {
		return fmt.Errorf("error downloading initial page: %w", err)
	}

	return nil
}

func createDirectory(baseURL string) (string, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	domain := parsedURL.Host
	err = os.MkdirAll(domain, 0o755)
	return domain, err
}

func downloadPage(pageURL, baseFolder string, reject, exclude []string, convertLinks bool, visited map[string]bool) error {
	if visited[pageURL] {
		return nil
	}
	visited[pageURL] = true

	resp, err := http.Get(pageURL)
	if err != nil {
		// Don't treat HTTP errors as fatal for linked pages
		fmt.Printf("Warning: Could not access %s: %v\n", pageURL, err)
		return nil
	}
	defer resp.Body.Close()

	// Handle non-200 status codes
	if resp.StatusCode != http.StatusOK {
		// For the initial page (baseURL), we want to return an error
		if len(visited) == 1 {
			return fmt.Errorf("failed to fetch initial page %s: status code %d", pageURL, resp.StatusCode)
		}
		// For linked pages, just log a warning and continue
		fmt.Printf("Warning: Page %s returned status code %d\n", pageURL, resp.StatusCode)
		return nil
	}

	// Check if the content is actually HTML before processing
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "text/html") {
		fmt.Printf("Skipping non-HTML content at %s (Content-Type: %s)\n", pageURL, contentType)
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	relativePath, err := getRelativePath(pageURL, baseFolder)
	if err != nil {
		return fmt.Errorf("error getting relative path: %w", err)
	}

	for _, excludeDir := range exclude {
		if strings.HasPrefix(relativePath, excludeDir) {
			return nil
		}
	}

	htmlPath := filepath.Join(baseFolder, relativePath)
	err = os.MkdirAll(filepath.Dir(htmlPath), 0o755)
	if err != nil {
		return fmt.Errorf("error creating directories: %w", err)
	}

	htmlContent := string(body)
	if convertLinks {
		htmlContent = convertLinksInHTML(htmlContent, pageURL, baseFolder)
	}

	err = os.WriteFile(htmlPath, []byte(htmlContent), 0o644)
	if err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	fmt.Printf("Successfully downloaded: %s\n", relativePath)

	return downloadResources(htmlContent, pageURL, baseFolder, reject, exclude, convertLinks, visited)
}

func downloadResources(htmlContent, pageURL, baseFolder string, reject, exclude []string, convertLinks bool, visited map[string]bool) error {
    // Updated regex to specifically target a, link, and img tags
    resourceRegex := regexp.MustCompile(`(?i)<(a|link|img)[^>]+(?:href|src)="([^"]+)"`)
    var lastErr error

    for _, match := range resourceRegex.FindAllStringSubmatch(htmlContent, -1) {
        tagName := strings.ToLower(match[1])
        resourceURL := match[2]

        // Skip empty URLs or fragment-only URLs
        if resourceURL == "" || strings.HasPrefix(resourceURL, "#") {
            continue
        }

        // Skip javascript: URLs
        if strings.HasPrefix(resourceURL, "javascript:") {
            continue
        }

        absoluteURL := resolveURL(pageURL, resourceURL)
        if absoluteURL == "" {
            fmt.Printf("Warning: Could not resolve URL: %s\n", resourceURL)
            continue
        }

        if shouldReject(absoluteURL, reject) {
            fmt.Printf("Skipping rejected URL: %s\n", absoluteURL)
            continue
        }

        // Only download resources from the same domain
        baseURLParsed, err := url.Parse(pageURL)
        if err != nil {
            fmt.Printf("Warning: Could not parse base URL: %s\n", pageURL)
            continue
        }

        resourceURLParsed, err := url.Parse(absoluteURL)
        if err != nil {
            fmt.Printf("Warning: Could not parse resource URL: %s\n", absoluteURL)
            continue
        }

        if resourceURLParsed.Host != baseURLParsed.Host {
            fmt.Printf("Skipping external resource: %s\n", absoluteURL)
            continue
        }

        // Handle differently based on tag type
        switch tagName {
        case "a":
            // For <a> tags, only follow if it's an HTML page
            if strings.HasSuffix(strings.ToLower(resourceURL), ".html") || 
               !strings.Contains(resourceURL, ".") {
                err := downloadPage(absoluteURL, baseFolder, reject, exclude, convertLinks, visited)
                if err != nil {
                    fmt.Printf("Warning: Error downloading linked page %s: %v\n", absoluteURL, err)
                    lastErr = err
                }
            }
        case "link", "img":
            // For <link> and <img> tags, download as resources
            err := downloadFile(absoluteURL, baseFolder)
            if err != nil {
                fmt.Printf("Warning: Error downloading resource %s: %v\n", absoluteURL, err)
                lastErr = err
            }
        }
    }

    return lastErr
}

func downloadFile(fileURL, baseFolder string) error {
	resp, err := http.Get(fileURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received status code %d", resp.StatusCode)
	}

	relativePath, err := getRelativePath(fileURL, baseFolder)
	if err != nil {
		return err
	}

	filePath := filepath.Join(baseFolder, relativePath)
	err = os.MkdirAll(filepath.Dir(filePath), 0o755)
	if err != nil {
		return err
	}

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	fmt.Printf("Downloaded: %s\n", relativePath)
	return nil
}

func resolveURL(baseURL, resourceURL string) string {
	if strings.HasPrefix(resourceURL, "http") {
		return resourceURL
	}
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	resolvedURL := parsedBaseURL.ResolveReference(&url.URL{Path: resourceURL})
	return resolvedURL.String()
}

func getRelativePath(resourceURL, baseFolder string) (string, error) {
	parsedURL, err := url.Parse(resourceURL)
	if err != nil {
		return "", err
	}

	path := parsedURL.Path
	if path == "" {
		path = "/"
	}

	// Handle trailing slash by adding index.html
	if strings.HasSuffix(path, "/") {
		path += "index.html"
	} else {
		// Check if the path has no extension
		ext := filepath.Ext(path)
		if ext == "" {
			path += ".html"
		}
	}

	// Clean the path to handle any ".." or "." components
	path = filepath.Clean(path)

	// Ensure the path starts with a separator
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path, nil
}

func shouldReject(resourceURL string, reject []string) bool {
	for _, suffix := range reject {
		if strings.HasSuffix(resourceURL, suffix) {
			return true
		}
	}
	return false
}

func convertLinksInHTML(htmlContent, pageURL, baseFolder string) string {
    // Match both href and src attributes in a, link, and img tags
    resourceRegex := regexp.MustCompile(`(?i)(<(?:a|link|img)[^>]+?(?:href|src)=")([^"]+)(")`);
    
    return resourceRegex.ReplaceAllStringFunc(htmlContent, func(match string) string {
        parts := resourceRegex.FindStringSubmatch(match)
        if len(parts) != 4 {
            return match
        }
        
        prefix := parts[1]      // The opening part of the tag until the URL
        resourceURL := parts[2] // The URL itself
        suffix := parts[3]      // The closing quote
        
        // Skip empty URLs, fragments, and javascript: URLs
        if resourceURL == "" || 
           strings.HasPrefix(resourceURL, "#") || 
           strings.HasPrefix(resourceURL, "javascript:") ||
           strings.HasPrefix(resourceURL, "data:") ||
           strings.HasPrefix(resourceURL, "mailto:") {
            return match
        }

        // Handle absolute and relative URLs
        absoluteURL := resolveURL(pageURL, resourceURL)
        if absoluteURL == "" {
            return match
        }

        // Only convert URLs from the same domain
        baseURLParsed, err := url.Parse(pageURL)
        if err != nil {
            return match
        }

        resourceURLParsed, err := url.Parse(absoluteURL)
        if err != nil {
            return match
        }

        if resourceURLParsed.Host != baseURLParsed.Host {
            return match // Keep external links unchanged
        }

        relativePath, err := getRelativePath(absoluteURL, baseFolder)
        if err != nil {
            return match
        }

        // Reconstruct the tag with the local path
        return prefix + relativePath + suffix
    })
}