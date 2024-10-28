package main

import (
	"path/filepath"
	"wget/utils"
)

func main() {
	output, url, path, background := utils.CheckFlags()

	filename := ""
	if output == "" {
		filename = utils.GetFileName(url)
	} else {
		filename = output
	}

	// Combine path and filename
	if path != "" {
		filename = filepath.Join(path, filename) 
	}

	utils.DownloadWithLogging(url, filename, background)
}
