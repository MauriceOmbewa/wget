package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	// Define flags
	outputFile := flag.String("O", "", "Specify the output filename")
	flag.Parse()

	// Get the URL from the remaining argument
	if flag.NArg() < 1 {
		fmt.Println("Usage: go run . [-O filename] <URL>")
		return
	}
	url := flag.Arg(0)

	// Handle output filename
	var fileName string
	if *outputFile != "" {
		fileName = *outputFile
	} else {
		fileName = "downloaded_file" // Default filename if -O is not provided
	}

	// Start the download process
	startTime := time.Now()
	fmt.Printf("start at %s\n", startTime.Format("2006-01-02 15:04:05"))

	// Perform the HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		//fmt.Printf("status %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
		fmt.Printf("Sending request, awaiting response... status %v\n", resp.StatusCode)
		return
	} else {
		fmt.Printf("Sending request, awaiting response... status %v OK\n", resp.StatusCode)
	}

	// Get content length
	contentLength := resp.ContentLength
	if contentLength < 0 {
		contentLength = 0
	}

	// Create and open the file
	file, err := os.Create(fileName)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	// Copy data from the response body to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		return
	}

	// Display file size
	contentLengthfloat := float64(contentLength)
	fmt.Printf("content size: %d [~%.2fMB]\n", contentLength, (float64(contentLengthfloat) / 1000000))
	fmt.Printf("saving file to: ./%s\n", fileName)

	//Displaying dynamic realtime progress bar and download speed
	// Example values
	totalSize := 55.05 * 1024            // Total size in KiB, converted to bytes
	chunkSize := 1024.0                  // Simulating download chunk size (in bytes)
	downloaded := 0.0                    // Bytes downloaded so far
	barLength := 100                     // Length of the progress bar
	startTime = time.Now()              // Start time to calculate elapsed time

	for downloaded < totalSize {
		time.Sleep(100 * time.Millisecond) // Simulate time delay per chunk
		downloaded += chunkSize
		if downloaded > totalSize {
			downloaded = totalSize
		}

		// Calculate percentage and download speed
		percentage := (downloaded / totalSize) * 100
		elapsed := time.Since(startTime).Seconds()
		speed := downloaded / elapsed / (1024 * 1024) // Speed in MiB/s

		// Create progress bar
		filledLength := int(percentage / 100 * float64(barLength))
		bar := strings.Repeat("=", filledLength) + strings.Repeat(" ", barLength-filledLength)

		// Print the progress line dynamically
		fmt.Printf("\r%.2f KiB / %.2f KiB [%s] %.2f%% %.2f MiB/s %.0fs",
			downloaded/1024, totalSize/1024, bar, percentage, speed, elapsed)
		if percentage == 100 {
			fmt.Println()
		}
	}
	fmt.Println()

	//Display downloaded url
	fmt.Printf("Downloaded [%s]\n", url)

	// End time
	endTime := time.Now()
	fmt.Printf("finished at %s\n", endTime.Format("2006-01-02 15:04:05"))
}
