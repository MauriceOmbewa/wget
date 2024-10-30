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

func downloadResources(htmlContent, pageURL, baseFolder string, reject, exclude []string, convertLinks bool, visited map[string]bool) error {
	resourceRegex, err := regexp.Compile(`(?i)<(a|link|img|script)[^>]+(?:href|src)="([^"]+)"`)
	if err != nil {
		return fmt.Errorf("failed to compile resource regex: %w", err)
	}

	var lastErr error

	// Process inline content first
	inlineImages, _, err := extractImagesFromInline(htmlContent, pageURL)
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
	// Compile regex to match href, src attributes, and inline style url() paths
	resourceRegex, err := regexp.Compile(`(?i)(<(?:a|link|img|script|style)[^>]+?(?:href|src)=")([^"]+)(")`)
	if err != nil {
		fmt.Printf("Warning: Failed to compile resource regex: %v\n", err)
		return htmlContent
	}

	urlInCSSRegex, err := regexp.Compile(`(?i)url\((['"]?)([^'")]+)['"]?\)`)
	if err != nil {
		fmt.Printf("Warning: Failed to compile CSS url regex: %v\n", err)
		return htmlContent
	}

	// Function to handle replacing src/href attributes
	replaceAttributeURLs := func(match string) string {
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

		if !strings.HasPrefix(relativePath, "./") {
			relativePath = "./" + relativePath
		}

		return prefix + relativePath + suffix
	}

	// Function to handle replacing url() paths within CSS
	replaceCSSURLs := func(cssContent string) string {
		return urlInCSSRegex.ReplaceAllStringFunc(cssContent, func(cssMatch string) string {
			parts := urlInCSSRegex.FindStringSubmatch(cssMatch)
			if len(parts) != 3 {
				return cssMatch
			}

			originalURL := parts[2]

			if strings.HasPrefix(originalURL, "#") ||
				strings.HasPrefix(originalURL, "javascript:") ||
				strings.HasPrefix(originalURL, "data:") ||
				strings.HasPrefix(originalURL, "mailto:") {
				return cssMatch
			}

			absoluteURL := resolveURL(pageURL, originalURL)
			if absoluteURL == "" {
				return cssMatch
			}

			relativePath, err := getRelativePath(absoluteURL, baseFolder)
			if err != nil {
				return cssMatch
			}

			if strings.HasPrefix(relativePath, "/") {
				relativePath = "." + relativePath
			} else if !strings.HasPrefix(relativePath, "/") && !strings.HasPrefix(relativePath, "./") {
				relativePath = "./" + relativePath
			}

			return "url('" + relativePath + "')"
		})
	}

	// Replace attribute URLs in HTML content
	modifiedContent := resourceRegex.ReplaceAllStringFunc(htmlContent, replaceAttributeURLs)

	// Replace URLs within inline styles and <style> tags
	styleTagRegex, err := regexp.Compile(`(<style[^>]*>)([\s\S]*?)(</style>)`)
	if err != nil {
		fmt.Printf("Warning: Failed to compile style tag regex: %v\n", err)
		return modifiedContent
	}

	modifiedContent = styleTagRegex.ReplaceAllStringFunc(modifiedContent, func(styleMatch string) string {
		parts := styleTagRegex.FindStringSubmatch(styleMatch)
		if len(parts) != 4 {
			return styleMatch
		}

		openTag := parts[1]
		cssContent := parts[2]
		closeTag := parts[3]

		// Update URLs within CSS content
		modifiedCSSContent := replaceCSSURLs(cssContent)
		return openTag + modifiedCSSContent + closeTag
	})

	return modifiedContent
}

func notConvertLinksInHTML(htmlContent, pageURL, baseFolder string) string {
    // Compile regex to match href, src attributes, and inline style url() paths
    resourceRegex, err := regexp.Compile(`(?i)(<(?:a|link|img|script|style)[^>]+?(?:href|src)=")([^"]+)(")`)
    if err != nil {
        fmt.Printf("Warning: Failed to compile resource regex: %v\n", err)
        return htmlContent
    }

    urlInCSSRegex, err := regexp.Compile(`(?i)url\((['"]?)([^'")]+)['"]?\)`)
    if err != nil {
        fmt.Printf("Warning: Failed to compile CSS url regex: %v\n", err)
        return htmlContent
    }

    // Function to handle replacing src/href attributes
    replaceAttributeURLs := func(match string) string {
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

        if !strings.HasPrefix(relativePath, "./") {
            relativePath = "./" + relativePath
        }

        return prefix + relativePath + suffix
    }

    // Function to handle replacing url() paths within CSS
    replaceCSSURLs := func(cssContent string) string {
        return urlInCSSRegex.ReplaceAllStringFunc(cssContent, func(cssMatch string) string {
            parts := urlInCSSRegex.FindStringSubmatch(cssMatch)
            if len(parts) != 3 {
                return cssMatch
            }

            originalURL := parts[2]

            if strings.HasPrefix(originalURL, "#") ||
                strings.HasPrefix(originalURL, "javascript:") ||
                strings.HasPrefix(originalURL, "data:") ||
                strings.HasPrefix(originalURL, "mailto:") {
                return cssMatch
            }

            absoluteURL := resolveURL(pageURL, originalURL)
            if absoluteURL == "" {
                return cssMatch
            }

            relativePath, err := getRelativePath(absoluteURL, baseFolder)
            if err != nil {
                return cssMatch
            }

            if strings.HasPrefix(relativePath, "/") {
                relativePath = "." + relativePath
            } else if !strings.HasPrefix(relativePath, "/") && !strings.HasPrefix(relativePath, "./") {
                relativePath = "./" + relativePath
            }

            return "url('" + relativePath + "')"
        })
    }

    // Replace attribute URLs in HTML content
    modifiedContent := resourceRegex.ReplaceAllStringFunc(htmlContent, replaceAttributeURLs)

    // Replace URLs within inline styles and <style> tags
    styleTagRegex, err := regexp.Compile(`(<style[^>]*>)([\s\S]*?)(</style>)`)
    if err != nil {
        fmt.Printf("Warning: Failed to compile style tag regex: %v\n", err)
        return modifiedContent
    }

    modifiedContent = styleTagRegex.ReplaceAllStringFunc(modifiedContent, func(styleMatch string) string {
        parts := styleTagRegex.FindStringSubmatch(styleMatch)
        if len(parts) != 4 {
            return styleMatch
        }

        openTag := parts[1]
        cssContent := parts[2]
        closeTag := parts[3]

        // Update URLs within CSS content
        modifiedCSSContent := replaceCSSURLs(cssContent)
        return openTag + modifiedCSSContent + closeTag
    })

    return modifiedContent
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
	} else if !convertLinks{
		htmlContent = notConvertLinksInHTML(htmlContent, pageURL, baseFolder)
	}

	// absoluteURL := resolveURL(pageURL, resourceURL)
	// if absoluteURL == "" {
	// 	return match
	// }

	// relativePath, err = getRelativePath(absoluteURL, baseFolder)
	// if err != nil {
	// 	return err
	// }

	returnValue := downloadResources(htmlContent, pageURL, baseFolder, reject, exclude, convertLinks, visited)

	err = os.WriteFile(htmlPath, []byte(htmlContent), 0o644)
	if err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	fmt.Printf("Successfully downloaded: %s\n", relativePath)
	
	return returnValue
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
    if path == "" || strings.HasSuffix(path, "/") {
        path = filepath.Join(path, "index.html")
    } else {
        ext := filepath.Ext(path)
        if ext == "" {
            path += ".html"
        }
    }

    // Clean the path to handle any ".." or "." components
    cleanPath := filepath.Clean(path)

    // Convert URL path to file system path
    filePath := filepath.Join(baseFolder, cleanPath)

    // Ensure the path is relative to the base folder
    relativePath, err := filepath.Rel(baseFolder, filePath)
    if err != nil {
        return "", err
    }

    // Ensure relativePath uses forward slashes for URLs
    relativePath = filepath.ToSlash(relativePath)

    // Prepend "./" if the path does not start with "." or "/"
    if !strings.HasPrefix(relativePath, "./") && !strings.HasPrefix(relativePath, "/") {
        relativePath = "./" + relativePath
    }

    return relativePath, nil
}

func shouldReject(resourceURL string, reject []string) bool {
	for _, suffix := range reject {
		if strings.HasSuffix(resourceURL, suffix) {
			return true
		}
	}
	return false
}

func extractImagesFromCSS(cssContent, baseURL string) ([]string, string, error) {
    var images []string
    urlRegex, err := regexp.Compile(`url\(['"]?([^'"()]+)['"]?\)`)
    if err != nil {
        return nil, cssContent, fmt.Errorf("failed to compile CSS URL regex: %w", err)
    }

    modifiedContent := cssContent
    matches := urlRegex.FindAllStringSubmatch(cssContent, -1)
    matchIndices := urlRegex.FindAllStringSubmatchIndex(cssContent, -1)

    // Process matches in reverse order to avoid offset issues
    for i := len(matches) - 1; i >= 0; i-- {
        match := matches[i]
        indices := matchIndices[i]

        if len(match) >= 2 {
            path := match[1]  // The captured path group

            // Skip data URLs and external URLs
            if !strings.HasPrefix(path, "data:") && !strings.HasPrefix(path, "http") {
                images = append(images, path)

                // Create the new path with dot prefix
                newPath := path
                if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, ".") {
                    newPath = "./" + path
                } else if strings.HasPrefix(path, "/") {
                    newPath = "." + path
                }

                // Create the new url() statement
                newFullMatch := fmt.Sprintf("url('%s')", newPath)

                // Replace in the content using the correct indices
                start := indices[0]
                end := indices[1]
                modifiedContent = modifiedContent[:start] + newFullMatch + modifiedContent[end:]
            }
        }
    }

    return images, modifiedContent, nil
}

func extractImagesFromJS(jsContent, baseURL string) ([]string, string, error) {
    var images []string
    modifiedContent := jsContent

    patterns := []string{
        `(['"])([^'"]+\.(?:jpg|jpeg|png|gif|svg|webp))(['"])`,
        `(['"])([^'"]+\/images\/[^'"]+)(['"])`,
        `(['"])([^'"]+\/img\/[^'"]+)(['"])`,
    }

    for _, pattern := range patterns {
        regex, err := regexp.Compile(pattern)
        if err != nil {
            return nil, modifiedContent, fmt.Errorf("failed to compile JS pattern regex: %w", err)
        }

        matches := regex.FindAllStringSubmatch(modifiedContent, -1)
        matchIndices := regex.FindAllStringSubmatchIndex(modifiedContent, -1)

        // Process matches in reverse order to avoid offset issues
        for i := len(matches) - 1; i >= 0; i-- {
            match := matches[i]
            indices := matchIndices[i]
            
            if len(match) >= 4 {
                quote := match[1]  // Preserve the original quote style
                path := match[2]

                if !strings.HasPrefix(path, "data:") && !strings.HasPrefix(path, "http") {
                    images = append(images, path)

                    // Create the new path with dot prefix
                    newPath := path
                    if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, ".") {
                        newPath = "./" + path
                    } else if strings.HasPrefix(path, "/") {
                        newPath = "." + path
                    }

                    // Replace in the content using the correct indices
                    start := indices[0]
                    end := indices[1]
                    modifiedContent = modifiedContent[:start] + quote + newPath + quote + modifiedContent[end:]
                }
            }
        }
    }

    return images, modifiedContent, nil
}

func extractImagesFromHTML(htmlContent, baseURL string) ([]string, string, error) {
    var images []string
    modifiedContent := htmlContent

    // Updated regex to capture the quotes and full img tag
    patterns := []string{
        `(<img[^>]+src=)(['"])([^'"]+\.(?:jpg|jpeg|png|gif|svg|webp))(['"])([^>]*>)`,
        `(<img[^>]+src=)(['"])([^'"]+\/images\/[^'"]+)(['"])([^>]*>)`,
        `(<img[^>]+src=)(['"])([^'"]+\/img\/[^'"]+)(['"])([^>]*>)`,
    }

    for _, pattern := range patterns {
        regex, err := regexp.Compile(pattern)
        if err != nil {
            return nil, modifiedContent, fmt.Errorf("failed to compile HTML pattern regex: %w", err)
        }

        matches := regex.FindAllStringSubmatch(modifiedContent, -1)
        matchIndices := regex.FindAllStringSubmatchIndex(modifiedContent, -1)

        // Process matches in reverse order to avoid offset issues
        for i := len(matches) - 1; i >= 0; i-- {
            match := matches[i]
            indices := matchIndices[i]

            if len(match) >= 4 {
                srcPrefix := match[1]    // <img src=
                quote := match[2]        // quote character
                path := match[3]         // the actual path
                restOfTag := match[5]    // rest of the img tag

                if !strings.HasPrefix(path, "data:") && !strings.HasPrefix(path, "http") {
                    images = append(images, path)

                    // Create the new path with dot prefix
                    newPath := path
                    if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, ".") {
                        newPath = "./" + path
                    } else if strings.HasPrefix(path, "/") {
                        newPath = "." + path
                    }

                    // Replace in the content using the correct indices
                    start := indices[0]
                    end := indices[1]
                    modifiedContent = modifiedContent[:start] + srcPrefix + quote + newPath + quote + restOfTag + modifiedContent[end:]
                }
            }
        }
    }

    return images, modifiedContent, nil
}

// 3. Fix extractImagesFromInline to ensure consistent path handling in style blocks
func extractImagesFromInline(htmlContent, baseURL string) ([]string, string, error) {
    var images []string
    modifiedContent := htmlContent

    // Extract and process inline styles
    styleRegex, err := regexp.Compile(`(<style[^>]*>)([\s\S]*?)(</style>)`)
    if err != nil {
        return nil, modifiedContent, fmt.Errorf("failed to compile style regex: %w", err)
    }

    styleMatches := styleRegex.FindAllStringSubmatchIndex(modifiedContent, -1)
    for i := len(styleMatches) - 1; i >= 0; i-- {
        match := styleMatches[i]
        if len(match) >= 8 {
            openTag := modifiedContent[match[2]:match[3]]
            cssContent := modifiedContent[match[4]:match[5]]
            closeTag := modifiedContent[match[6]:match[7]]

            cssImages, modifiedCSS, err := extractImagesFromCSS(cssContent, baseURL)
            if err != nil {
                return nil, modifiedContent, fmt.Errorf("failed to extract CSS images: %w", err)
            }
            images = append(images, cssImages...)

            // Replace the content between style tags with modified content
            modifiedContent = modifiedContent[:match[0]] + openTag + modifiedCSS + closeTag + modifiedContent[match[1]:]
        }
    }

    // Similar updates for script and HTML blocks...
    scriptRegex, err := regexp.Compile(`(<script[^>]*>)([\s\S]*?)(</script>)`)
    if err != nil {
        return nil, modifiedContent, fmt.Errorf("failed to compile script regex: %w", err)
    }

    scriptMatches := scriptRegex.FindAllStringSubmatchIndex(modifiedContent, -1)
    for i := len(scriptMatches) - 1; i >= 0; i-- {
        match := scriptMatches[i]
        if len(match) >= 8 {
            openTag := modifiedContent[match[2]:match[3]]
            jsContent := modifiedContent[match[4]:match[5]]
            closeTag := modifiedContent[match[6]:match[7]]

            jsImages, modifiedJS, err := extractImagesFromJS(jsContent, baseURL)
            if err != nil {
                return nil, modifiedContent, fmt.Errorf("failed to extract JS images: %w", err)
            }
            images = append(images, jsImages...)

            // Replace the content between script tags with modified content
            modifiedContent = modifiedContent[:match[0]] + openTag + modifiedJS + closeTag + modifiedContent[match[1]:]
        }
    }

    return images, modifiedContent, nil
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

	// Extract images and get modified content
	images, modifiedContent, err := extractImagesFromCSS(string(content), pageURL)
	if err != nil {
		return fmt.Errorf("failed to extract images from CSS: %w", err)
	}

	// Download images
	for _, imgPath := range images {
		absoluteURL := resolveURL(cssURL, imgPath)
		if absoluteURL != "" {
			err := downloadFile(absoluteURL, baseFolder)
			if err != nil {
				fmt.Printf("Warning: Error downloading CSS image %s: %v\n", absoluteURL, err)
			}
		}
	}

	// Save modified CSS content
	relativePath, err := getRelativePath(cssURL, baseFolder)
	if err != nil {
		return fmt.Errorf("failed to get relative path for CSS: %w", err)
	}

	filePath := filepath.Join(baseFolder, relativePath)
	err = os.MkdirAll(filepath.Dir(filePath), 0o755)
	if err != nil {
		return fmt.Errorf("failed to create CSS directory: %w", err)
	}

	err = os.WriteFile(filePath, []byte(modifiedContent), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write CSS file: %w", err)
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

	// Extract images and get modified content
	images, modifiedContent, err := extractImagesFromJS(string(content), pageURL)
	if err != nil {
		return fmt.Errorf("failed to extract images from JS: %w", err)
	}

	// Download images
	for _, imgPath := range images {
		absoluteURL := resolveURL(jsURL, imgPath)
		if absoluteURL != "" {
			err := downloadFile(absoluteURL, baseFolder)
			if err != nil {
				fmt.Printf("Warning: Error downloading JS image %s: %v\n", absoluteURL, err)
			}
		}
	}

	// Save modified JS content
	relativePath, err := getRelativePath(jsURL, baseFolder)
	if err != nil {
		return fmt.Errorf("failed to get relative path for JS: %w", err)
	}

	filePath := filepath.Join(baseFolder, relativePath)
	err = os.MkdirAll(filepath.Dir(filePath), 0o755)
	if err != nil {
		return fmt.Errorf("failed to create JS directory: %w", err)
	}

	err = os.WriteFile(filePath, []byte(modifiedContent), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write JS file: %w", err)
	}

	return nil
}
