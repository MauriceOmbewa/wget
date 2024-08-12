package main

import (
	"wget/utils"
)

func main() {
	output, url, background := utils.CheckFlags()

	filename := ""
	if output == "" {
		filename = utils.GetFileName(url)
	} else {
		filename = output
	}

	utils.DownloadWithLogging(url, filename, background)
}
