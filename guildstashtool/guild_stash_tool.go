package guildstashtool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	RateLimiter *RateLimiter
	SessionId   string
	BplJwt      string
	GuildId     string
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

func (c *Client) getLatestTimestamp() (int64, error) {
	url := "https://v2202503259898322516.goodsrv.de/api/current/guild-stash/history/latest_timestamp"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.BplJwt))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var latestTimestamp int64
	json.Unmarshal(body, &latestTimestamp)
	return latestTimestamp, nil
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
		return 0, latestId, fmt.Errorf("HttpStatusCode: %d (Guild not found - Guild ID or PoE Session ID invalid)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, latestId, fmt.Errorf("HttpStatusCode: %d (PoE Session ID most likely invalid)", resp.StatusCode)
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
	fmt.Printf("Requesting stashes from %v to %v - received %d entries\n", time.Unix(end, 0).Format(time.DateTime), time.Unix(start, 0).Format(time.DateTime), len(unmarshalled.Entries))

	if len(unmarshalled.Entries) == 0 || !unmarshalled.Truncated {
		return 0, latestId, nil
	}
	go c.sendStashHistoryToBplBackend(body)
	lastEntry := unmarshalled.Entries[len(unmarshalled.Entries)-1]
	return c.getHistoryBetween(lastEntry.Time, end, lastEntry.Id)
}

func (c *Client) sendStashHistoryToBplBackend(body []byte) {
	url := "https://v2202503259898322516.goodsrv.de/api/current/guild-stash/history"

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
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return
	}
}

// RunStashMonitoring runs the guild stash monitoring once
func RunStashMonitoring(sessionId, bplJwt, guildId string) error {
	client := NewClient(sessionId, bplJwt, guildId)

	latestTimestamp, err := client.getLatestTimestamp()
	if err != nil {
		return fmt.Errorf("error getting latest timestamp: %w", err)
	}

	now := time.Now().Unix()
	_, _, err = client.getHistoryBetween(now, latestTimestamp, "")
	if err != nil {
		return fmt.Errorf("error getting history: %w", err)
	}

	fmt.Println("Guild stash monitoring completed successfully")
	return nil
}

// RunStashMonitoringContinuous runs the guild stash monitoring continuously
func RunStashMonitoringContinuous(sessionId, bplJwt, guildId string, interval time.Duration) {
	client := NewClient(sessionId, bplJwt, guildId)

	latestTimestamp, err := client.getLatestTimestamp()
	if err != nil {
		fmt.Printf("Error getting latest timestamp: %v\n", err)
		return
	}

	for {
		now := time.Now().Unix()
		_, _, err = client.getHistoryBetween(now, latestTimestamp, "")
		latestTimestamp = now
		if err != nil {
			fmt.Printf("Error getting history: %v\n", err)
			return
		}
		time.Sleep(interval)
	}
}
