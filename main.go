package main

import (
	"log"
	"wget/utils"
)

func main() {
	output, url, background, file, rateLimit := utils.CheckFlags()

	// Handle multi-file download case
	if file != "" {
		urls, err := utils.ReadUrlsFromFile(file)
		if err != nil {
			log.Fatal(err)
		}
		
		// Pass rate limit to concurrent download function
		err = utils.DownloadFilesConcurrently(urls, output, background, rateLimit)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	// Handle single file download case
	filename := output
	if filename == "" {
		filename = utils.GetFileName(url)
	}

	utils.DownloadWithLogging(url, filename, background, rateLimit)
}