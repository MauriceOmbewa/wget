package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func DownloadFile(urlStr, fileName string, background bool) error {
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
	fmt.Printf("content size: %d [~%.2fMB]\n", contentLength, float64(contentLength)/1000/1000)

	// Save file
	out, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}
	defer out.Close()

	fmt.Printf("saving file to: ./%s\n", fileName)

	if background {
		_, err := io.Copy(out, resp.Body)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}
	} else {
		bar := NewProgressBar(contentLength, 50)
		bar.StartTimer()

		_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}
	}
	// End time
	endTime := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("Downloaded [%s]\nfinished at %s\n", urlStr, endTime)

	return nil
}

func DownloadWithLogging(urlStr string, fileName string, background bool) {
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

			err1 := DownloadFile(urlStr, fileName, background)
			if err1 != nil {
				fmt.Fprintln(logFile, "Error:", err)
			}
		}()
		// Prevent the main function from exiting immediately
		time.Sleep(3 * time.Second)
	} else {
		err := DownloadFile(urlStr, fileName, background)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func GetFileName(url string) string {
	s := strings.Split(url, "/")

	// Extract the file name from the URL path
	return s[len(s)-1]
}
