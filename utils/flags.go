package utils

import (
	"flag"
	"fmt"
)

func CheckFlags() (output, url string, tolog bool) {
	outputFile := flag.String("O", "", "Specify the output filename")
	log := flag.Bool("B", false, "Run download in the background")
	flag.Parse()

	// Get the URL from the remaining argument
	if flag.NArg() < 1 {
		fmt.Println("Usage: go run . [-O filename] <URL>")
		return
	}
	url = flag.Arg(0)
	return *outputFile, url, *log
}
