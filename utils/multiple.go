// function to download multiple files concurrently
package utils

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// function to read urls from a file
func ReadUrlsFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var urls []string
	for scanner.Scan() {
		urls = append(urls, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	fmt.Println(urls)
	return urls, nil
}

func DownloadFilesConcurrently(urls []string, outputPrefix string, background bool, rateLimit int64, path string) error {
	var wg sync.WaitGroup
	errorChan := make(chan error, len(urls))

	// If rate limit is specified, divide it among concurrent downloads
	var perFileRateLimit int64
	if rateLimit > 0 {
		perFileRateLimit = rateLimit / int64(len(urls))
		fmt.Printf("Rate limit per file: %.2f KB/s\n", float64(perFileRateLimit)/1024)
	}

	// Print total content size
	sizes := make([]int64, len(urls))
	for i, url := range urls {
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("error getting content size: %v", err)
		}
		sizes[i] = resp.ContentLength
		resp.Body.Close()
	}
	fmt.Printf("Content size: %v\n", sizes)

	for i, url := range urls {
		wg.Add(1)
		go func(url string, index int) {
			defer wg.Done()

			var filename string
			if outputPrefix != "" {
				filename = fmt.Sprintf("%s_%d", outputPrefix, index)
			} else {
				filename = GetFileName(url)
			}

			// Combine path with filename if path is specified
			if path != "" {
				filename = filepath.Join(path, filename)
			}

			err := DownloadFile(url, filename, background, perFileRateLimit)
			if err != nil {
				errorChan <- fmt.Errorf("error downloading %s: %v", url, err)
				return
			}
			fmt.Printf("Finished %s\n", filename)
		}(url, i)
	}

	// Wait for all downloads to complete
	go func() {
		wg.Wait()
		close(errorChan)
	}()

	// Check for any errors
	var errCount int
	for err := range errorChan {
		errCount++
		fmt.Println(err)
	}

	if errCount > 0 {
		return fmt.Errorf("%d downloads failed", errCount)
	}

	fmt.Printf("\nDownload finished: %v\n", urls)
	return nil
}
