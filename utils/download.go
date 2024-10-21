package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func DownloadFile(urlStr, fileName string, background bool, rateLimit int64) error {
    startTime := time.Now().Format("2006-01-02 15:04:05")
    fmt.Printf("start at %s\n", startTime)

    // Create an HTTP client
    client := &http.Client{}

    // Create a new request
    req, err := http.NewRequest("GET", urlStr, nil)
    if err != nil {
        return fmt.Errorf("error creating request: %v", err)
    }

    // Set a User-Agent header to simulate a browser
    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")

    // Send the request
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("error: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("error: got status %s", resp.Status)
    }
    fmt.Printf("sending request, awaiting response... status %s\n", resp.Status)

    contentLength := resp.ContentLength
    fmt.Printf("content size: %d [~%.2fMB]\n", contentLength, float64(contentLength)/1000/1000)

    out, err := os.Create(fileName)
    if err != nil {
        return fmt.Errorf("error: %v", err)
    }
    defer out.Close()

    fmt.Printf("saving file to: ./%s\n", fileName)

    // Create rate-limited reader if rate limit is specified else download normal
    var reader io.Reader = resp.Body
    if rateLimit > 0 {
        fmt.Printf("Rate limit set to: %.2f KB/s\n", float64(rateLimit)/1024)
        reader = NewRateLimitReader(resp.Body, rateLimit)
    }

    if background {
        _, err := io.Copy(out, reader)
        if err != nil {
            return fmt.Errorf("error: %v", err)
        }
    } else {
        bar := NewProgressBar(contentLength, 50)
        bar.StartTimer()

        _, err = io.Copy(io.MultiWriter(out, bar), reader)
        if err != nil {
            return fmt.Errorf("error: %v", err)
        }
    }

    endTime := time.Now().Format("2006-01-02 15:04:05")
    fmt.Printf("Downloaded [%s]\nfinished at %s\n", urlStr, endTime)

    return nil
}


func DownloadWithLogging(urlStr string, fileName string, background bool, rateLimit int64) {
	if background {
		fmt.Println("Output will be written to 'wget-log'.")
		go func() {
			logFile, err := os.Create("wget-log")
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			defer logFile.Close()

			os.Stdout = logFile
			os.Stderr = logFile

			err1 := DownloadFile(urlStr, fileName, background, rateLimit)
			if err1 != nil {
				fmt.Fprintln(logFile, "Error:", err)
			}
		}()
		time.Sleep(3 * time.Second)
	} else {
		err := DownloadFile(urlStr, fileName, background, rateLimit)
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
