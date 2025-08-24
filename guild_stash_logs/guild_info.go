package guild_stash_logs

import (
	"fmt"
	htmlutil "html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// GuildInfo represents the parsed guild information
type GuildInfo struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

// FetchGuildInfo fetches and parses guild information from the PoE website
func FetchGuildInfo(sessionID string) (*GuildInfo, error) {
	url := "https://www.pathofexile.com/my-guild"

	// Create HTTP client
	client := &http.Client{}

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers to mimic browser request
	req.Header.Add("user-agent", "Contact: liberatorist@gmail.com")
	req.Header.Add("Cookie", fmt.Sprintf("POESESSID=%s", sessionID))
	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse HTML and extract guild info
	guildInfo, err := parseGuildInfo(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse guild info: %w", err)
	}

	return guildInfo, nil
}

// parseGuildInfo extracts guild information from HTML content
func parseGuildInfo(htmlContent string) (*GuildInfo, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	guildInfo := &GuildInfo{}

	// Extract guild ID from tab links (e.g., /guild/profile/408208)
	guildIDRegex := regexp.MustCompile(`/guild/profile/(\d+)`)
	matches := guildIDRegex.FindStringSubmatch(htmlContent)
	if len(matches) > 1 {
		guildInfo.Id, err = strconv.Atoi(matches[1])
		if err != nil {
			return nil, fmt.Errorf("failed to convert guild ID: %w", err)
		}
	}

	// Extract guild name and tag using HTML parsing
	var walkHTML func(*html.Node)
	walkHTML = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Look for guild name in h1 with class "name"
			if n.Data == "h1" {
				for _, attr := range n.Attr {
					if attr.Key == "class" && attr.Val == "name" {
						if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
							guildInfo.Name = strings.TrimSpace(n.FirstChild.Data)
						}
					}
				}
			}

			// Look for guild tag in p with class "guild-tag"
			if n.Data == "p" {
				for _, attr := range n.Attr {
					if attr.Key == "class" && attr.Val == "guild-tag" {
						if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
							tag := strings.TrimSpace(n.FirstChild.Data)
							// Decode HTML entities and remove surrounding brackets
							tag = htmlutil.UnescapeString(tag)
							tag = strings.Trim(tag, "<>")
							guildInfo.Tag = tag
						}
					}
				}
			}
		}

		// Recursively walk through child nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkHTML(c)
		}
	}

	walkHTML(doc)

	// Validate that we found the required information
	if guildInfo.Name == "" {
		return nil, fmt.Errorf("guild name not found in HTML")
	}

	if guildInfo.Tag == "" {
		return nil, fmt.Errorf("guild tag not found in HTML")
	}

	if guildInfo.Id == 0 {
		return nil, fmt.Errorf("guild ID not found in HTML")
	}

	return guildInfo, nil
}
