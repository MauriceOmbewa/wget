package utils

import (
	"fmt"
	"strings"
	"time"
)


type ProgressBar struct {
	Total     int64
	Written   int64
	StartTime time.Time
	BarLength int
}

// an io writer 
func (pb *ProgressBar) Write(p []byte) (int, error) {
	n := len(p)
	pb.Written += int64(n)
	pb.printProgress()
	return n, nil
}

// start time of the download
func (pb *ProgressBar) StartTimer() {
	pb.StartTime = time.Now()
}

// total time taken for the download.
func (pb *ProgressBar) EndTimer() time.Duration {
	return time.Since(pb.StartTime)
}

// Calculate the speed of copying in bytes per second.
func (pb *ProgressBar) CalculateSpeed() float64 {
	duration := time.Since(pb.StartTime).Seconds()
	if duration == 0 {
		return 0
	}
	return float64(pb.Written) / duration
}

// display the progress bar, percentage, and speed.
func (pb *ProgressBar) printProgress() {
	since:=pb.EndTimer()
	downloaded:=float64(pb.Written)/1000
	total:=float64(pb.Total)/1000
	percent := float64(pb.Written) / float64(pb.Total) * 100
	filledLength := int(percent) * pb.BarLength / 100
	bar := strings.Repeat("#", filledLength) + strings.Repeat(" ", pb.BarLength-filledLength)
	speed := pb.CalculateSpeed()/1000/1000
	fmt.Printf("\r %.2f KiB / %.2f KiB [%s] %.2f%% | %.2f MB/s %.0fs", downloaded,total,bar, percent, speed, since.Seconds())
	if pb.Written == pb.Total {
		fmt.Print("\n\n")
	}
}

// initialize a new ProgressBar.
func NewProgressBar(total int64, barLength int) *ProgressBar {
	return &ProgressBar{
		Total:     total,
		BarLength: barLength,
	}
}
