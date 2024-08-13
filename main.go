package main

import (
	"wget/utils"
)

func main() {
output, url,path, background := utils.CheckFlags()

	filename := ""
	if output == "" {
		filename = utils.GetFileName(url)
	} else {
		filename = output
	}
if path !=""{
	filename = path + filename
}
	utils.DownloadWithLogging(url, filename, background)
}
