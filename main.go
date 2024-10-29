package main

import (
	"log"
	"path/filepath"
	"wget/utils"
)

func main() {
	output, url, background, file, rateLimit, mirror, reject, exclude, convertLinks, path := utils.CheckFlags()

	if mirror {
		// Handle mirroring
		if url == "" {
			log.Fatal("URL is required for mirroring")
		}
		err := utils.MirrorWebsite(url, reject, exclude, convertLinks)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	// Handle multi-file download case
	if file != "" {
		urls, err := utils.ReadUrlsFromFile(file)
		if err != nil {
			log.Fatal(err)
		}
		// Pass rate limit and output directory to concurrent download function
		err = utils.DownloadFilesConcurrently(urls, output, background, rateLimit, path)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	// Handle single file download case
	if url == "" {
		log.Fatal("URL is required for single file download")
	}

	filename := output
	if filename == "" {
		filename = utils.GetFileName(url)
	}

	// Combine path and filename if path is specified
	if path != "" {
		filename = filepath.Join(path, filename)
	}

	utils.DownloadWithLogging(url, filename, background, rateLimit)
}
