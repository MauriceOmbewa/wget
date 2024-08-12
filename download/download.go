package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

func downloadFile(urlStr, fileName string) error {
	// Start time
	startTime := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("start at %s\n", startTime)

	// Send HTTP GET request
	resp, err := http.Get(urlStr)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}
	defer resp.Body.Close()

	// Check for HTTP status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error: got status %s", resp.Status)
	}
	fmt.Printf("sending request, awaiting response... status %s\n", resp.Status)

	// Get the content length
	contentLength := resp.ContentLength
	fmt.Printf("content size: %d [~%.2fMB]\n", contentLength, float64(contentLength)/1024/1024)

	// Save file
	out, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}
	defer out.Close()

	fmt.Printf("saving file to: ./%s\n", fileName)

	// Copy data from response to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}

	// End time
	endTime := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("Downloaded [%s]\nfinished at %s\n", urlStr, endTime)

	return nil
}

func downloadWithLogging(urlStr string, background bool) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		fmt.Printf("Error parsing URL: %v\n", err)
		return
	}

	// Extract the file name from the URL path
	fileName := path.Base(parsedURL.Path)

	if background {
		fmt.Println("Output will be written to 'wget-log'.")
		go func() {
			logFile, err := os.Create("wget-log")
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			defer logFile.Close()

			// Redirect output to the log file
			os.Stdout = logFile
			os.Stderr = logFile

			err = downloadFile(urlStr, fileName)
			if err != nil {
				fmt.Fprintln(logFile, "Error:", err)
			}
		}()
		// Prevent the main function from exiting immediately
		time.Sleep(1 * time.Second)
	} else {
		err := downloadFile(urlStr, fileName)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func main() {
	// Define the -B flag
	background := flag.Bool("B", false, "Run download in the background")

	// Parse the command-line flags
	flag.Parse()

	// Ensure a URL is provided
	if flag.NArg() < 1 {
		fmt.Println("Usage: [options] <url>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// The first non-flag argument is the URL
	urlStr := flag.Arg(0)

	// Call the download function with the parsed flag
	downloadWithLogging(urlStr, *background)
}
