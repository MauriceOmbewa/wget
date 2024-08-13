package utils

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func CheckFlags() (output, url string, filepath string, tolog bool) {
	outputFile := flag.String("O", "", "Specify the output filename")
	path := flag.String("P", "", "Specify the file path")
	log := flag.Bool("B", false, "Run download in the background")
	flag.Parse()

	// Get the URL from the remaining argument
	if flag.NArg() < 1 {
		fmt.Println("Usage: go run . [-O filename] <URL>")
		return
	}
	url = flag.Arg(0)
	if strings.Contains(*path, "~") {
		home := os.Getenv("HOME")
		*path = strings.Replace(*path, "~", home, 1)
		*path = strings.TrimSpace(*path)
		if !strings.HasPrefix(*path, "/") {
			*path = "/" + *path
		}
	}
	return *outputFile, url, *path, *log
}
