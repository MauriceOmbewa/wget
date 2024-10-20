package utils

import (
	"flag"
	"fmt"
)

func CheckFlags() (output, url string, tolog bool, file string, rateLimit int64) {
	outputFile := flag.String("O", "", "Specify the output filename")
	log := flag.Bool("B", false, "Run download in the background")
	inputFile := flag.String("i", "", "Download multiple files from a list of URLs")
	rateLimitFlag := flag.String("rate-limit", "", "Limit download speed (e.g., 400k, 2M)")
	flag.Parse()

	if *inputFile == "" {
		if flag.NArg() < 1 {
			fmt.Println("Usage: go run . [-O filename] [-B] [-i urlfile] [--rate-limit rate] <URL>")
			fmt.Println("  -O filename     : Specify output filename")
			fmt.Println("  -B              : Run download in background")
			fmt.Println("  -i urlfile      : Download multiple URLs from file")
			fmt.Println("  --rate-limit rate: Limit download speed (e.g., 400k, 2M)")
			return
		}
		url = flag.Arg(0)
	}

	limit, err := ParseRateLimit(*rateLimitFlag)
	if err != nil {
		fmt.Printf("Warning: Invalid rate limit format: %v\n", err)
	}

	return *outputFile, url, *log, *inputFile, limit
}
