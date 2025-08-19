package guildstashtool

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RateLimiter struct {
	mutex       sync.Mutex
	rateLimits  []RateLimit
	lastUpdated time.Time
}

type RateLimit struct {
	maxRequests   int
	windowSeconds int
	resetSeconds  int
	currentUsage  int
	windowStart   time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		rateLimits: []RateLimit{
			// Default fallback limits if headers are not available
			{maxRequests: 60, windowSeconds: 60, resetSeconds: 60},
			{maxRequests: 120, windowSeconds: 300, resetSeconds: 300},
			{maxRequests: 300, windowSeconds: 3600, resetSeconds: 3600},
		},
		lastUpdated: time.Now(),
	}
}

// UpdateFromResponse updates rate limits from PoE API response headers
func (rl *RateLimiter) UpdateFromResponse(resp *http.Response) error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Parse X-Rate-Limit-Account header: "60:60:10,120:300:300,300:3600:1800"
	rateLimitHeader := resp.Header.Get("X-Rate-Limit-Account")
	if rateLimitHeader == "" {
		return fmt.Errorf("no X-Rate-Limit-Account header found")
	}

	// Parse X-Rate-Limit-Account-State header: "1:60:0,1:300:0,18:3600:0"
	stateHeader := resp.Header.Get("X-Rate-Limit-Account-State")
	if stateHeader == "" {
		return fmt.Errorf("no X-Rate-Limit-Account-State header found")
	}

	now := time.Now()

	// Parse rate limits
	limitParts := strings.Split(rateLimitHeader, ",")
	stateParts := strings.Split(stateHeader, ",")

	if len(limitParts) != len(stateParts) {
		return fmt.Errorf("mismatch between rate limit and state headers")
	}

	newLimits := make([]RateLimit, 0, len(limitParts))

	for i, limitPart := range limitParts {
		// Parse limit: "60:60:10" -> max_requests:window_seconds:reset_seconds
		limitValues := strings.Split(limitPart, ":")
		if len(limitValues) != 3 {
			continue
		}

		// Parse state: "1:60:0" -> current_requests:window_seconds:seconds_until_reset
		stateValues := strings.Split(stateParts[i], ":")
		if len(stateValues) != 3 {
			continue
		}

		maxRequests, _ := strconv.Atoi(limitValues[0])
		windowSeconds, _ := strconv.Atoi(limitValues[1])
		resetSeconds, _ := strconv.Atoi(limitValues[2])

		currentUsage, _ := strconv.Atoi(stateValues[0])
		// stateValues[1] should match windowSeconds
		secondsUntilReset, _ := strconv.Atoi(stateValues[2])

		// Calculate when this window started
		// If secondsUntilReset is 0, the window just reset, so start from now
		var windowStart time.Time
		if secondsUntilReset == 0 {
			windowStart = now
		} else {
			windowStart = now.Add(-time.Duration(windowSeconds-secondsUntilReset) * time.Second)
		}

		newLimits = append(newLimits, RateLimit{
			maxRequests:   maxRequests,
			windowSeconds: windowSeconds,
			resetSeconds:  resetSeconds,
			currentUsage:  currentUsage,
			windowStart:   windowStart,
		})
	}

	if len(newLimits) > 0 {
		rl.rateLimits = newLimits
		rl.lastUpdated = now
	}

	return nil
}

// Wait blocks until it's safe to make a request according to current rate limits
func (rl *RateLimiter) Wait() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	var maxWait time.Duration

	for _, limit := range rl.rateLimits {
		// Calculate how much time is left in this window
		windowEnd := limit.windowStart.Add(time.Duration(limit.windowSeconds) * time.Second)
		timeLeftInWindow := windowEnd.Sub(now)

		// If the window has expired, reset the usage
		if timeLeftInWindow <= 0 {
			limit.currentUsage = 0
			limit.windowStart = now
			continue
		}

		// If we're at or over the limit, we need to wait
		if limit.currentUsage >= limit.maxRequests {
			if timeLeftInWindow > maxWait {
				maxWait = timeLeftInWindow
			}
		}
	}

	// If we need to wait, do so
	if maxWait > 0 {
		time.Sleep(maxWait)

		// After waiting, reset any expired windows
		now = time.Now()
		for i := range rl.rateLimits {
			windowEnd := rl.rateLimits[i].windowStart.Add(time.Duration(rl.rateLimits[i].windowSeconds) * time.Second)
			if now.After(windowEnd) {
				rl.rateLimits[i].currentUsage = 0
				rl.rateLimits[i].windowStart = now
			}
		}
	}

	// Increment usage for all limits (since we're about to make a request)
	for i := range rl.rateLimits {
		rl.rateLimits[i].currentUsage++
	}
}
