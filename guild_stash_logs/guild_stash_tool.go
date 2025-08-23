package guild_stash_logs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var bplBaseUrl = "https://v2202503259898322516.goodsrv.de/api"

// var bplBaseUrl = "http://localhost:8000/api"

// CredentialError represents an error related to invalid credentials
type CredentialError struct {
	Type    string // "poe_session", "bpl_token", "guild_id"
	Message string
	Code    int
}

func (e *CredentialError) Error() string {
	return e.Message
}

func NewCredentialError(credType, message string, code int) *CredentialError {
	return &CredentialError{
		Type:    credType,
		Message: message,
		Code:    code,
	}
}

type Client struct {
	RateLimiter    *RateLimiter
	SessionId      string
	BplJwt         string
	GuildId        string
	leagueStart    int64
	leagueEnd      int64
	rateLimitState string
}

type GuildStashChangeResponse struct {
	Entries []struct {
		Id      string `json:"id"`
		Time    int64  `json:"time"`
		League  string `json:"league"`
		Stash   string `json:"stash"`
		Item    string `json:"item"`
		Action  string `json:"action"`
		Account struct {
			Name string `json:"name"`
		} `json:"account"`
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"entries"`
	Truncated bool `json:"truncated"`
}

func NewClient(sessionId, bplJwt, guildId string) *Client {
	return &Client{
		RateLimiter: NewRateLimiter(),
		SessionId:   sessionId,
		BplJwt:      bplJwt,
		GuildId:     guildId,
	}
}

func (c *Client) getTimestamps() (*GuildStashLogTimestampResponse, error) {
	url := fmt.Sprintf("%s/current/guilds/%s/stash-history/latest_timestamp", bplBaseUrl, c.GuildId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.BplJwt))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return nil, NewCredentialError("bpl_token", fmt.Sprintf("HttpStatusCode: %d (BPL Token invalid or expired)", res.StatusCode), res.StatusCode)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HttpStatusCode: %d from BPL backend", res.StatusCode)
	}

	body, _ := io.ReadAll(res.Body)
	var timestamps GuildStashLogTimestampResponse
	if err := json.Unmarshal(body, &timestamps); err != nil {
		return nil, err
	}
	return &timestamps, nil
}

func (c *Client) updateProgress(message string, currentTimestamp int64) {
	// Clear the current lines and move cursor up
	fmt.Print("\033[2K\r")        // Clear current line
	fmt.Print("\033[1A\033[2K\r") // Move up and clear previous line

	// Display rate limiter state
	if c.RateLimiter != nil {
		fmt.Printf("Rate Limiter: %s\n", c.RateLimiter.GetState())
	}

	if c.leagueStart != 0 && c.leagueEnd != 0 {
		totalDuration := c.leagueEnd - c.leagueStart
		// Since we go backwards from end to start, calculate remaining duration
		remainingDuration := currentTimestamp - c.leagueStart
		progressDuration := totalDuration - remainingDuration

		// Ensure we don't go over 100% or below 0%
		if progressDuration > totalDuration {
			progressDuration = totalDuration
		}
		if progressDuration < 0 {
			progressDuration = 0
		}

		percentage := int((progressDuration * 100) / totalDuration)
		bar := strings.Repeat("=", percentage/2)
		spaces := strings.Repeat(" ", 50-len(bar))

		currentTime := time.Unix(currentTimestamp, 0).Format("2006-01-02 15:04")
		fmt.Printf("%s [%s%s] %d%% (Current: %s)",
			message, bar, spaces, percentage, currentTime)
	} else {
		fmt.Printf("%s...", message)
	}
}

func (c *Client) getHistoryBetween(start int64, end int64, startId string) (newStart int64, latestId string, err error) {
	url := fmt.Sprintf("https://www.pathofexile.com/api/guild/%s/stash/history?from=%d&end=%d", c.GuildId, end, start)
	if startId != "" {
		url += fmt.Sprintf("&fromid=%s", startId)
	}
	c.RateLimiter.Wait()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, latestId, err
	}

	req.Header.Add("user-agent", "Contact: liberatorist@gmail.com")
	req.Header.Add("Cookie", fmt.Sprintf("POESESSID=%s", c.SessionId))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, latestId, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return 0, latestId, NewCredentialError("guild_id", fmt.Sprintf("HttpStatusCode: %d (Guild not found - Guild ID invalid or no access rights)", resp.StatusCode), resp.StatusCode)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		retry, err := strconv.Atoi(resp.Header.Get("retry-after"))
		if err != nil {
			return 0, latestId, fmt.Errorf("HttpStatusCode: %d (Too many requests - Wait 30m before trying again)", resp.StatusCode)
		}
		duration := time.Duration(retry) * time.Second
		return 0, latestId, fmt.Errorf("HttpStatusCode: %d (Too many requests - Wait %v before trying again)", resp.StatusCode, duration)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return 0, latestId, NewCredentialError("poe_session", fmt.Sprintf("HttpStatusCode: %d (PoE Session ID most likely invalid)", resp.StatusCode), resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, latestId, NewCredentialError("poe_session", fmt.Sprintf("HttpStatusCode: %d (PoE Session ID most likely invalid)", resp.StatusCode), resp.StatusCode)
	}

	if updateErr := c.RateLimiter.UpdateFromResponse(resp); updateErr != nil {
		fmt.Printf("Warning: Could not update rate limiter: %v\n", updateErr)
	}

	body, _ := io.ReadAll(resp.Body)
	unmarshalled := GuildStashChangeResponse{}
	err = json.Unmarshal(body, &unmarshalled)
	if err != nil {
		return 0, latestId, err
	}

	if len(unmarshalled.Entries) > 0 {
		// Use the timestamp of the first entry to show progress
		firstEntryTimestamp := unmarshalled.Entries[0].Time
		c.updateProgress("Processing stash history", firstEntryTimestamp)
	}

	if len(unmarshalled.Entries) == 0 || !unmarshalled.Truncated {
		return 0, latestId, nil
	}
	go c.sendStashHistoryToBplBackend(body)
	lastEntry := unmarshalled.Entries[len(unmarshalled.Entries)-1]
	return c.getHistoryBetween(lastEntry.Time, end, lastEntry.Id)
}

func (c *Client) sendStashHistoryToBplBackend(body []byte) {
	url := fmt.Sprintf("%s/current/guilds/%s/stash-history", bplBaseUrl, c.GuildId)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.BplJwt))
	req.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("Error status from BPL backend: %v\n", resp.Status)
		return
	}
	var addResponse AddGuildStashHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResponse); err != nil {
		fmt.Printf("Error reading add response: %v\n", err)
		return
	}
}

func RunStashMonitoring(sessionId, bplJwt, guildId string) error {
	client := NewClient(sessionId, bplJwt, guildId)

	fmt.Print("Fetching timestamps...")
	timestamps, err := client.getTimestamps()
	if err != nil {
		return fmt.Errorf("error getting latest timestamp: %w", err)
	}
	fmt.Print("\rTimestamps fetched successfully\n")

	// Set league times for progress calculation
	client.leagueStart = timestamps.LeagueStart
	client.leagueEnd = timestamps.LeagueEnd

	dayAfterLeagueEnd := timestamps.LeagueEnd + 24*60*60
	if timestamps.Earliest == nil || timestamps.Latest == nil {
		_, _, err = client.getHistoryBetween(dayAfterLeagueEnd, timestamps.LeagueStart, "")
	} else {
		_, _, err = client.getHistoryBetween(*timestamps.Earliest, timestamps.LeagueStart, "")
		if err != nil {
			return fmt.Errorf("error getting history: %w", err)
		}
		_, _, err = client.getHistoryBetween(dayAfterLeagueEnd, *timestamps.Latest, "")
	}
	if err != nil {
		return fmt.Errorf("error getting history: %w", err)
	}
	fmt.Print("\rGuild stash monitoring completed successfully          \n")
	return nil
}

func RunStashMonitoringContinuous(sessionId, bplJwt, guildId string, interval time.Duration) error {
	client := NewClient(sessionId, bplJwt, guildId)

	timestamps, err := client.getTimestamps()
	if err != nil {
		return err
	}
	dayAfterLeagueEnd := timestamps.LeagueEnd + 24*60*60
	if timestamps.Earliest == nil || timestamps.Latest == nil {
		_, _, err = client.getHistoryBetween(dayAfterLeagueEnd, timestamps.LeagueStart, "")
	} else {
		_, _, err = client.getHistoryBetween(*timestamps.Earliest, timestamps.LeagueStart, "")
	}
	if err != nil {
		return err
	}

	for {
		_, _, err = client.getHistoryBetween(dayAfterLeagueEnd, *timestamps.Latest, "")
		now := time.Now().Unix()
		timestamps.Latest = &now
		if err != nil {
			return err
		}
		time.Sleep(interval)
	}
}

type AddGuildStashHistoryResponse struct {
	NumberOfAddedEntries int `json:"number_of_added_entries"`
}

type GuildStashLogTimestampResponse struct {
	Earliest    *int64 `json:"earliest"`
	Latest      *int64 `json:"latest"`
	LeagueStart int64  `json:"league_start"`
	LeagueEnd   int64  `json:"league_end"`
}
