package utils

import (
	"flag"
	"fmt"
	"strings"
)

func CheckFlags() (output, url string, tolog bool, file string, rateLimit int64, mirror bool, reject, exclude []string, convertLinks bool) {
	outputFile := flag.String("O", "", "Specify the output filename")
	log := flag.Bool("B", false, "Run download in the background")
	inputFile := flag.String("i", "", "Download multiple files from a list of URLs")
	rateLimitFlag := flag.String("rate-limit", "", "Limit download speed (e.g., 400k, 2M)")

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
			fmt.Println("Usage: go run . [-O filename] [-B] [-i urlfile] [--rate-limit rate] [--mirror] [-R suffixes] [-X directories] [--convert-links] <URL>")
			fmt.Println("  -O filename      : Specify output filename")
			fmt.Println("  -B               : Run download in background")
			fmt.Println("  -i urlfile       : Download multiple URLs from file")
			fmt.Println("  --rate-limit rate: Limit download speed (e.g., 400k, 2M)")
			fmt.Println("  --mirror         : Mirror the entire website")
			fmt.Println("  -R, --reject     : Reject file suffixes (comma-separated)")
			fmt.Println("  -X, --exclude    : Exclude directories (comma-separated)")
			fmt.Println("  --convert-links  : Convert links for offline viewing")
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

	// Remove empty strings from reject and exclude slices
	reject = removeEmptyStrings(reject)
	exclude = removeEmptyStrings(exclude)

	return *outputFile, url, *log, *inputFile, limit, *mirrorFlag, reject, exclude, *convertLinksFlag
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