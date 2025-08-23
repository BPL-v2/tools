package guild_stash_logs

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Policy struct {
	MaxHits int
	Period  time.Duration
}

func (p *Policy) CurrentHits(requestTimes []time.Time) int {
	periodStart := time.Now().Add(-p.Period)
	count := 0
	for _, t := range requestTimes {
		if t.After(periodStart) {
			count++
		}
	}
	return count
}

func (p *Policy) IsViolated(requestTimes []time.Time) bool {
	return p.CurrentHits(requestTimes) >= p.MaxHits
}

type RateLimiter struct {
	mutex        sync.Mutex
	policies     []Policy
	requestTimes []time.Time
	lastUpdated  time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		policies: []Policy{
			// Default fallback limits if headers are not available
			{MaxHits: 60, Period: 60 * time.Second},
			{MaxHits: 120, Period: 300 * time.Second},
			{MaxHits: 300, Period: 3600 * time.Second},
		},
		requestTimes: make([]time.Time, 0),
		lastUpdated:  time.Now(),
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

	newPolicies := make([]Policy, 0, len(limitParts))

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
		currentUsage, _ := strconv.Atoi(stateValues[0])

		policy := Policy{
			MaxHits: maxRequests,
			Period:  time.Duration(windowSeconds) * time.Second,
		}
		newPolicies = append(newPolicies, policy)

		// Add missing timestamps to match server's reported usage
		trackedHits := policy.CurrentHits(rl.requestTimes)
		missingHits := currentUsage - trackedHits
		for j := 0; j < missingHits; j++ {
			rl.requestTimes = append(rl.requestTimes, now)
		}
	}

	if len(newPolicies) > 0 {
		rl.policies = newPolicies
		rl.lastUpdated = now
	}

	// Clean old timestamps (older than 10 minutes)
	rl.cleanExpiredRequests(now)

	return nil
} // cleanExpiredRequests removes requests that are older than 10 minutes
func (rl *RateLimiter) cleanExpiredRequests(now time.Time) {
	cutoff := now.Add(-600 * time.Second) // 10 minutes

	validRequests := make([]time.Time, 0, len(rl.requestTimes))
	for _, reqTime := range rl.requestTimes {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}

	rl.requestTimes = validRequests
}

// canMakeRequest checks if a request can be made without violating any rate limit
func (rl *RateLimiter) canMakeRequest() bool {
	for _, policy := range rl.policies {
		if policy.IsViolated(rl.requestTimes) {
			return false
		}
	}
	return true
}

// Wait blocks until it's safe to make a request according to current rate limits
func (rl *RateLimiter) Wait() {
	for {
		rl.mutex.Lock()
		canMake := rl.canMakeRequest()
		if canMake {
			// Record this request
			now := time.Now()
			rl.requestTimes = append(rl.requestTimes, now)
			rl.cleanExpiredRequests(now)
			rl.mutex.Unlock()
			return
		}
		rl.mutex.Unlock()

		// Wait a short time before checking again
		time.Sleep(100 * time.Millisecond)
	}
}

func (rl *RateLimiter) GetState() string {
	states := make([]string, len(rl.policies))
	for i, policy := range rl.policies {
		states[i] = fmt.Sprintf("%d:%d:%d", policy.CurrentHits(rl.requestTimes), policy.MaxHits, int(math.Round(policy.Period.Seconds())))
	}
	return strings.Join(states, ", ")
}
