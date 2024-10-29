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
	// "bytes"
)

// Main function remains unchanged
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

func extractImagesFromCSS(cssContent, baseURL string) ([]string, error) {
	var images []string
	
	urlRegex, err := regexp.Compile(`url\(['"]?([^'"()]+)['"]?\)`)
	if err != nil {
		return nil, fmt.Errorf("failed to compile CSS URL regex: %w", err)
	}
	
	matches := urlRegex.FindAllStringSubmatch(cssContent, -1)
	
	for _, match := range matches {
		if len(match) >= 2 {
			path := match[1]
			// Skip data URLs and external URLs
			if !strings.HasPrefix(path, "data:") && !strings.HasPrefix(path, "http") {
				// Add dot prefix if path doesn't start with / or .
				if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, ".") {
					path = "./" + path
				}
				images = append(images, path)
			}
		}
	}
	
	return images, nil
}

func extractImagesFromJS(jsContent, baseURL string) ([]string, error) {
	var images []string
	
	patterns := []string{
		`['"]([^'"]+\.(?:jpg|jpeg|png|gif|svg|webp))['"]`,
		`['"]([^'"]+\/images\/[^'"]+)['"]`,
		`['"]([^'"]+\/img\/[^'"]+)['"]`,
	}
	
	for _, pattern := range patterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to compile JS pattern regex: %w", err)
		}
		
		matches := regex.FindAllStringSubmatch(jsContent, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				path := match[1]
				if !strings.HasPrefix(path, "data:") && !strings.HasPrefix(path, "http") {
					if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, ".") {
						path = "./" + path
					}
					images = append(images, path)
				}
			}
		}
	}
	
	return images, nil
}

func extractImagesFromInline(htmlContent, baseURL string) ([]string, error) {
	var images []string
	
	// Extract inline styles
	styleRegex, err := regexp.Compile(`<style[^>]*>([\s\S]*?)</style>`)
	if err != nil {
		return nil, fmt.Errorf("failed to compile style regex: %w", err)
	}
	
	styleMatches := styleRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range styleMatches {
		if len(match) >= 2 {
			cssImages, err := extractImagesFromCSS(match[1], baseURL)
			if err != nil {
				return nil, fmt.Errorf("failed to extract CSS images: %w", err)
			}
			images = append(images, cssImages...)
		}
	}
	
	// Extract inline scripts
	scriptRegex, err := regexp.Compile(`<script[^>]*>([\s\S]*?)</script>`)
	if err != nil {
		return nil, fmt.Errorf("failed to compile script regex: %w", err)
	}
	
	scriptMatches := scriptRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range scriptMatches {
		if len(match) >= 2 {
			jsImages, err := extractImagesFromJS(match[1], baseURL)
			if err != nil {
				return nil, fmt.Errorf("failed to extract JS images: %w", err)
			}
			images = append(images, jsImages...)
		}
	}
	
	return images, nil
}

func downloadResources(htmlContent, pageURL, baseFolder string, reject, exclude []string, convertLinks bool, visited map[string]bool) error {
	resourceRegex, err := regexp.Compile(`(?i)<(a|link|img|script)[^>]+(?:href|src)="([^"]+)"`)
	if err != nil {
		return fmt.Errorf("failed to compile resource regex: %w", err)
	}
	
	var lastErr error
	
	// Process inline content first
	inlineImages, err := extractImagesFromInline(htmlContent, pageURL)
	if err != nil {
		fmt.Printf("Warning: Error extracting inline images: %v\n", err)
		lastErr = err
	}
	
	for _, imgPath := range inlineImages {
		absoluteURL := resolveURL(pageURL, imgPath)
		if absoluteURL != "" {
			err := downloadFile(absoluteURL, baseFolder)
			if err != nil {
				fmt.Printf("Warning: Error downloading inline image %s: %v\n", absoluteURL, err)
				lastErr = err
			}
		}
	}

	baseURLParsed, err := url.Parse(pageURL)
	if err != nil {
		return fmt.Errorf("failed to parse base URL: %w", err)
	}

	for _, match := range resourceRegex.FindAllStringSubmatch(htmlContent, -1) {
		tagName := strings.ToLower(match[1])
		resourceURL := match[2]

		if resourceURL == "" || 
		   strings.HasPrefix(resourceURL, "#") || 
		   strings.HasPrefix(resourceURL, "javascript:") ||
		   strings.HasPrefix(resourceURL, "data:") {
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

		resourceURLParsed, err := url.Parse(absoluteURL)
		if err != nil {
			fmt.Printf("Warning: Could not parse resource URL %s: %v\n", absoluteURL, err)
			continue
		}

		// Check if the resource is from the same domain or its subdomain
		if !isSameOrSubdomain(baseURLParsed.Host, resourceURLParsed.Host) {
			fmt.Printf("Skipping external resource: %s\n", absoluteURL)
			continue
		}

		switch tagName {
		case "a":
			if strings.HasSuffix(strings.ToLower(resourceURL), ".html") || 
			   !strings.Contains(resourceURL, ".") {
				err := downloadPage(absoluteURL, baseFolder, reject, exclude, convertLinks, visited)
				if err != nil {
					fmt.Printf("Warning: Error downloading linked page %s: %v\n", absoluteURL, err)
					lastErr = err
				}
			}
		case "link":
			if strings.HasSuffix(strings.ToLower(resourceURL), ".css") {
				err := downloadAndProcessCSS(absoluteURL, baseFolder, pageURL)
				if err != nil {
					fmt.Printf("Warning: Error processing CSS %s: %v\n", absoluteURL, err)
					lastErr = err
				}
			}
		case "script":
			if strings.HasSuffix(strings.ToLower(resourceURL), ".js") {
				err := downloadAndProcessJS(absoluteURL, baseFolder, pageURL)
				if err != nil {
					fmt.Printf("Warning: Error processing JS %s: %v\n", absoluteURL, err)
					lastErr = err
				}
			}
		case "img":
			err := downloadFile(absoluteURL, baseFolder)
			if err != nil {
				fmt.Printf("Warning: Error downloading resource %s: %v\n", absoluteURL, err)
				lastErr = err
			}
		}
	}

	return lastErr
}

func downloadAndProcessCSS(cssURL, baseFolder, pageURL string) error {
	resp, err := http.Get(cssURL)
	if err != nil {
		return fmt.Errorf("failed to download CSS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received status code %d for CSS", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read CSS content: %w", err)
	}

	relativePath, err := getRelativePath(cssURL, baseFolder)
	if err != nil {
		return fmt.Errorf("failed to get relative path for CSS: %w", err)
	}

	filePath := filepath.Join(baseFolder, relativePath)
	err = os.MkdirAll(filepath.Dir(filePath), 0o755)
	if err != nil {
		return fmt.Errorf("failed to create CSS directory: %w", err)
	}

	err = os.WriteFile(filePath, content, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write CSS file: %w", err)
	}

	images, err := extractImagesFromCSS(string(content), pageURL)
	if err != nil {
		return fmt.Errorf("failed to extract images from CSS: %w", err)
	}

	for _, imgPath := range images {
		absoluteURL := resolveURL(cssURL, imgPath)
		if absoluteURL != "" {
			err := downloadFile(absoluteURL, baseFolder)
			if err != nil {
				fmt.Printf("Warning: Error downloading CSS image %s: %v\n", absoluteURL, err)
			}
		}
	}

	return nil
}

func downloadAndProcessJS(jsURL, baseFolder, pageURL string) error {
	resp, err := http.Get(jsURL)
	if err != nil {
		return fmt.Errorf("failed to download JS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received status code %d for JS", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read JS content: %w", err)
	}

	relativePath, err := getRelativePath(jsURL, baseFolder)
	if err != nil {
		return fmt.Errorf("failed to get relative path for JS: %w", err)
	}

	filePath := filepath.Join(baseFolder, relativePath)
	err = os.MkdirAll(filepath.Dir(filePath), 0o755)
	if err != nil {
		return fmt.Errorf("failed to create JS directory: %w", err)
	}

	err = os.WriteFile(filePath, content, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write JS file: %w", err)
	}

	images, err := extractImagesFromJS(string(content), pageURL)
	if err != nil {
		return fmt.Errorf("failed to extract images from JS: %w", err)
	}

	for _, imgPath := range images {
		absoluteURL := resolveURL(jsURL, imgPath)
		if absoluteURL != "" {
			err := downloadFile(absoluteURL, baseFolder)
			if err != nil {
				fmt.Printf("Warning: Error downloading JS image %s: %v\n", absoluteURL, err)
			}
		}
	}

	return nil
}

// New helper function to check if a host is the same domain or a subdomain
func isSameOrSubdomain(baseHost, resourceHost string) bool {
	baseHostParts := strings.Split(baseHost, ".")
	resourceHostParts := strings.Split(resourceHost, ".")
	
	if len(resourceHostParts) < len(baseHostParts) {
		return false
	}
	
	// Check if the base domain appears at the end of the resource domain
	for i := 1; i <= len(baseHostParts); i++ {
		if i > len(resourceHostParts) {
			return false
		}
		if baseHostParts[len(baseHostParts)-i] != resourceHostParts[len(resourceHostParts)-i] {
			return false
		}
	}
	
	return true
}

func convertLinksInHTML(htmlContent, pageURL, baseFolder string) string {
    resourceRegex, err := regexp.Compile(`(?i)(<(?:a|link|img|script)[^>]+?(?:href|src)=")([^"]+)(")`);
    if err != nil {
        fmt.Printf("Warning: Failed to compile resource regex: %v\n", err)
        return htmlContent
    }
    
    return resourceRegex.ReplaceAllStringFunc(htmlContent, func(match string) string {
        parts := resourceRegex.FindStringSubmatch(match)
        if len(parts) != 4 {
            return match
        }
        
        prefix := parts[1]
        resourceURL := parts[2]
        suffix := parts[3]
        
        if resourceURL == "" || 
           strings.HasPrefix(resourceURL, "#") || 
           strings.HasPrefix(resourceURL, "javascript:") ||
           strings.HasPrefix(resourceURL, "data:") ||
           strings.HasPrefix(resourceURL, "mailto:") {
            return match
        }

        absoluteURL := resolveURL(pageURL, resourceURL)
        if absoluteURL == "" {
            return match
        }

        baseURLParsed, err := url.Parse(pageURL)
        if err != nil {
            return match
        }

        resourceURLParsed, err := url.Parse(absoluteURL)
        if err != nil {
            return match
        }

        if !isSameOrSubdomain(baseURLParsed.Host, resourceURLParsed.Host) {
            return match
        }

        relativePath, err := getRelativePath(absoluteURL, baseFolder)
        if err != nil {
            return match
        }

        return prefix + relativePath + suffix
    })
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