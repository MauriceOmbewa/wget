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

//
func MirrorWebsite(baseURL string, reject []string, exclude []string, convertLinks bool) error {
	baseFolder, err := createDirectory(baseURL)
	if err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	visited := make(map[string]bool)
	err = downloadPage(baseURL, baseFolder, reject, exclude, convertLinks, visited)
	if err != nil {
		return fmt.Errorf("error downloading page: %w", err)
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
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch page: %s", pageURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	relativePath, err := getRelativePath(pageURL, baseFolder)
	if err != nil {
		return err
	}

	for _, excludeDir := range exclude {
		if strings.HasPrefix(relativePath, excludeDir) {
			return nil
		}
	}

	htmlPath := filepath.Join(baseFolder, relativePath)
	err = os.MkdirAll(filepath.Dir(htmlPath), 0o755)
	if err != nil {
		return err
	}

	htmlContent := string(body)
	if convertLinks {
		htmlContent = convertLinksInHTML(htmlContent, pageURL, baseFolder)
	}

	err = os.WriteFile(htmlPath, []byte(htmlContent), 0o644)
	if err != nil {
		return err
	}

	err = downloadResources(htmlContent, pageURL, baseFolder, reject, exclude, convertLinks, visited)
	if err != nil {
		return err
	}

	return nil
}

func downloadResources(htmlContent, pageURL, baseFolder string, reject, exclude []string, convertLinks bool, visited map[string]bool) error {
	resourceRegex := regexp.MustCompile(`(src|href)="(.*?)"`)

	for _, match := range resourceRegex.FindAllStringSubmatch(htmlContent, -1) {
		resourceURL := match[2]
		absoluteURL := resolveURL(pageURL, resourceURL)

		if shouldReject(absoluteURL, reject) {
			continue
		}

		if strings.HasPrefix(absoluteURL, pageURL) {
			err := downloadPage(absoluteURL, baseFolder, reject, exclude, convertLinks, visited)
			if err != nil {
				fmt.Printf("Error downloading linked page %s: %v\n", absoluteURL, err)
			}
		} else {
			err := downloadFile(absoluteURL, baseFolder)
			if err != nil {
				fmt.Printf("Error downloading resource %s: %v\n", absoluteURL, err)
			}
		}
	}

	return nil
}

func downloadFile(fileURL, baseFolder string) error {
	resp, err := http.Get(fileURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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
	relativePath := parsedURL.Path
	if relativePath == "" || strings.HasSuffix(relativePath, "/") {
		relativePath += "index.html"
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

func convertLinksInHTML(htmlContent, pageURL, baseFolder string) string {
	resourceRegex := regexp.MustCompile(`(src|href)="(.*?)"`)
	return resourceRegex.ReplaceAllStringFunc(htmlContent, func(match string) string {
		parts := resourceRegex.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		attr := parts[1]
		resourceURL := parts[2]
		absoluteURL := resolveURL(pageURL, resourceURL)
		relativePath, err := getRelativePath(absoluteURL, baseFolder)
		if err != nil {
			return match
		}
		return fmt.Sprintf(`%s="%s"`, attr, relativePath)
	})
}
