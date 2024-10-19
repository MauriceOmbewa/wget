package utils

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// RateLimitReader implements a rate-limited io.Reader
type RateLimitReader struct {
	reader     io.Reader
	rateLimit  int64 // in bytes per second for consistency
	lastRead   time.Time
	readBytes  int64
	bucketSize int64
}

// parseRateLimit converts rate which is a string (like "400k" or "2M") to bytes per second
func ParseRateLimit(rate string) (int64, error) {
	if rate == "" {
		return 0, nil
	}

	rate = strings.ToLower(rate)
	var multiplier int64 = 1

	switch {
	case strings.HasSuffix(rate, "k"):
		multiplier = 1024
		rate = rate[:len(rate)-1]
	case strings.HasSuffix(rate, "m"):
		multiplier = 1024 * 1024
		rate = rate[:len(rate)-1]
	}

	value, err := strconv.ParseInt(rate, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid rate limit format: %v", err)
	}

	return value * multiplier, nil
}

// NewRateLimitReader creates a new rate-limited reader
func NewRateLimitReader(reader io.Reader, rateLimit int64) *RateLimitReader {
	return &RateLimitReader{
		reader:     reader,
		rateLimit:  rateLimit,
		lastRead:   time.Now(),
		bucketSize: rateLimit, // Initial bucket size is equal to rate limit
	}
}

// Read is our custom io.Reader with rate limiting 
func (r *RateLimitReader) Read(p []byte) (int, error) {
	if r.rateLimit <= 0 {
		return r.reader.Read(p)
	}

	now := time.Now()
	duration := now.Sub(r.lastRead).Seconds()
	
	// Calculate how many bytes we can read based on the time passed
	r.bucketSize += int64(duration * float64(r.rateLimit))
	if r.bucketSize > r.rateLimit {
		r.bucketSize = r.rateLimit
	}

	// If we don't have enough in our bucket, sleep
	if r.bucketSize <= 0 {
		time.Sleep(time.Second / 10) // Sleep for a short duration
		return 0, nil
	}

	// Limit the read size to our available bucket size
	if int64(len(p)) > r.bucketSize {
		p = p[:r.bucketSize]
	}

	n, err := r.reader.Read(p)
	if n > 0 {
		r.bucketSize -= int64(n)
		r.readBytes += int64(n)
		r.lastRead = now
	}

	return n, err
}