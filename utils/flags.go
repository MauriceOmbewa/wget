package utils

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func CheckFlags() (output, url, filepath string, tolog bool) {
	outputFile := flag.String("O", "", "Specify the output filename")
	path := flag.String("P", "", "Specify the directory path for the download")
	log := flag.Bool("B", false, "Run download in the background")
	flag.Parse()

	// Get the URL from the remaining argument
	if flag.NArg() < 1 {
		fmt.Println("Usage: go run . [-O filename] [-P path] <URL>")
		return
	}
	url = flag.Arg(0)

	// Process and validate the path flag
	if *path != "" {
		// Expand "~" to the home directory
		if strings.HasPrefix(*path, "~") {
			home := os.Getenv("HOME")
			*path = strings.Replace(*path, "~", home, 1)
		}
		// Trim whitespace and trailing slashes
		*path = strings.TrimSpace(*path)
		if strings.HasSuffix(*path, "/") {
			*path = strings.TrimRight(*path, "/")
		}

		// Ensure the directory exists; create it if it doesn’t
		if err := os.MkdirAll(*path, os.ModePerm); err != nil {
			fmt.Printf("Failed to create directory %s: %v\n", *path, err)
			return
		}
	}

	return *outputFile, url, *path, *log
}
