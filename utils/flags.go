package utils

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func CheckFlags() (output, url string, tolog bool, file string, rateLimit int64, mirror bool, reject, exclude []string, convertLinks bool, path string) {
	outputFile := flag.String("O", "", "Specify the output filename")
	log := flag.Bool("B", false, "Run download in the background")
	inputFile := flag.String("i", "", "Download multiple files from a list of URLs")
	rateLimitFlag := flag.String("rate-limit", "", "Limit download speed (e.g., 400k, 2M)")
	pathFlag := flag.String("P", "", "Specify the directory path for downloads") // New path flag

	// New flags
	mirrorFlag := flag.Bool("mirror", false, "Mirror the entire website")
	rejectFlag := flag.String("R", "", "Reject file suffixes (comma-separated)")
	excludeFlag := flag.String("X", "", "Exclude directories (comma-separated)")
	convertLinksFlag := flag.Bool("convert-links", false, "Convert links for offline viewing")

	// Long-form versions of short flags
	flag.StringVar(rejectFlag, "reject", "", "Reject file suffixes (comma-separated)")
	flag.StringVar(excludeFlag, "exclude", "", "Exclude directories (comma-separated)")

	flag.Parse()

	if *inputFile == "" {
		if flag.NArg() < 1 && !*mirrorFlag {
			fmt.Println("Usage: go run . [-O filename] [-P path] [-B] [-i urlfile] [--rate-limit rate] [--mirror] [-R suffixes] [-X directories] [--convert-links] <URL>")
			return
		}
		if flag.NArg() > 0 {
			url = flag.Arg(0)
		}
	}

	limit, err := ParseRateLimit(*rateLimitFlag)
	if err != nil {
		fmt.Printf("Warning: Invalid rate limit format: %v\n", err)
	}

	// Process new flags
	reject = strings.Split(*rejectFlag, ",")
	exclude = strings.Split(*excludeFlag, ",")
	reject = removeEmptyStrings(reject)
	exclude = removeEmptyStrings(exclude)

	// Expand "~" in path if necessary
	if *pathFlag != "" && strings.HasPrefix(*pathFlag, "~") {
		home := os.Getenv("HOME")
		*pathFlag = strings.Replace(*pathFlag, "~", home, 1)
	}

	return *outputFile, url, *log, *inputFile, limit, *mirrorFlag, reject, exclude, *convertLinksFlag, *pathFlag
}

func removeEmptyStrings(s []string) []string {
	var result []string
	for _, str := range s {
		if str != "" {
			result = append(result, strings.TrimSpace(str))
		}
	}
	return result
}
