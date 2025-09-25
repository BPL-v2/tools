package league_invites

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// CredentialError represents an error related to invalid credentials
type CredentialError struct {
	Type    string // "poe_session", "bpl_token"
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

type Member struct {
	ID           int    `json:"id"`
	MemberName   string `json:"memberName"`
	Role         string `json:"role"`
	IsAcceptable bool   `json:"isAcceptable"`
}

type MembersResponse struct {
	Members []Member `json:"members"`
}

type Player struct {
	User   User `json:"user"`
	TeamID *int `json:"team_id"`
}

type User struct {
	AccountName string `json:"account_name"`
}

type AcceptRequest struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Client struct {
	Client          *http.Client
	PoeSessID       string
	BPLToken        string
	PrivateLeagueId string
	BPLUrl          string
}

type Event struct {
	Name string `json:"name"`
}

func NewClient(poeSessID, bplToken string) (*Client, error) {
	client := &Client{
		Client:    &http.Client{},
		PoeSessID: poeSessID,
		BPLToken:  bplToken,
		BPLUrl:    "https://v2202503259898322516.goodsrv.de/api",
	}
	err := client.setPrivateLeagueId()
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) setPrivateLeagueId() error {
	resp, err := c.Client.Get(c.BPLUrl + "/events/current")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return err
	}

	var leagueInfo Event
	if err := json.NewDecoder(resp.Body).Decode(&leagueInfo); err != nil {
		return err
	}
	regex := regexp.MustCompile(`\s*\(PL(\d+)\)`)
	matches := regex.FindStringSubmatch(leagueInfo.Name)
	if len(matches) != 2 {
		return fmt.Errorf("unexpected league name format: %s", leagueInfo.Name)
	}
	c.PrivateLeagueId = matches[1]
	fmt.Printf("Checking requests for league %s\n", leagueInfo.Name)
	return nil
}

func (c *Client) getLeagueJoinRequests() ([]Member, []Member, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://www.pathofexile.com/api/private-league-member/%s", c.PrivateLeagueId), nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("user-agent", "Liberatorist@gmail.com")
	req.AddCookie(&http.Cookie{Name: "POESESSID", Value: c.PoeSessID})

	q := req.URL.Query()
	q.Add("sort", "roleDesc")
	q.Add("search", "")
	q.Add("offset", "0")
	q.Add("limit", "100")
	q.Add("_", fmt.Sprintf("%d", time.Now().Unix()))
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, nil, NewCredentialError("poe_session", fmt.Sprintf("HttpStatusCode: %d (PoE Session ID invalid)", resp.StatusCode), resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("HttpStatusCode: %d from PoE API", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	var membersResp MembersResponse
	err = json.Unmarshal(body, &membersResp)
	if err != nil {
		return nil, nil, err
	}

	var requestedMembers []Member
	var acceptedMembers []Member
	for _, member := range membersResp.Members {
		if member.Role == "requested_invite" {
			requestedMembers = append(requestedMembers, member)
		} else {
			acceptedMembers = append(acceptedMembers, member)
		}
	}

	return requestedMembers, acceptedMembers, nil
}

func (c *Client) getSortedUsers() (map[string]bool, error) {
	req, err := http.NewRequest("GET", c.BPLUrl+"/events/current/signups", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.BPLToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, NewCredentialError("bpl_token", fmt.Sprintf("HttpStatusCode: %d (BPL Token invalid or expired)", resp.StatusCode), resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HttpStatusCode: %d from BPL API", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var players []Player
	err = json.Unmarshal(body, &players)
	if err != nil {
		return nil, err
	}

	sortedUsers := make(map[string]bool)
	for _, player := range players {
		if player.TeamID != nil {
			sortedUsers[player.User.AccountName] = true
		}
	}

	return sortedUsers, nil
}

func (c *Client) acceptPrivateLeagueInvites(members []Member) error {
	var acceptRequests []AcceptRequest
	for _, member := range members {
		acceptRequests = append(acceptRequests, AcceptRequest{
			Name:  "accept",
			Value: fmt.Sprintf("%d", member.ID),
		})
	}

	jsonData, err := json.Marshal(acceptRequests)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("https://www.pathofexile.com/api/private-league-member/%s", c.PrivateLeagueId), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Contact: Liberatorist@gmail.com")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.AddCookie(&http.Cookie{Name: "POESESSID", Value: c.PoeSessID})

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		return NewCredentialError("poe_session", fmt.Sprintf("HttpStatusCode: %d (PoE Session ID invalid) - Response: %s", resp.StatusCode, string(body)), resp.StatusCode)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to accept invites. Status code: %d, Response: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) HandlePrivateLeagueInvites() error {
	sortedUsers, err := c.getSortedUsers()
	if err != nil {
		return fmt.Errorf("failed to get sorted users: %w", err)
	}
	guildRequests, acceptedMembers, err := c.getLeagueJoinRequests()
	if err != nil {
		return fmt.Errorf("failed to get guild join requests: %w", err)
	}
	var membersToAdd []Member
	var unknownUsers []string

	fmt.Printf("Found %d requested invites and %d accepted members.\n", len(guildRequests), len(acceptedMembers))
	for _, member := range guildRequests {
		if sortedUsers[member.MemberName] && member.IsAcceptable {
			fmt.Printf("Accepting invite for user: %s\n", member.MemberName)
			membersToAdd = append(membersToAdd, member)
		} else if !sortedUsers[member.MemberName] {
			fmt.Printf("User %s is not sorted yet.\n", member.MemberName)
			unknownUsers = append(unknownUsers, member.MemberName)
		}
	}
	for _, member := range acceptedMembers {
		if !sortedUsers[member.MemberName] {
			fmt.Printf("User %s was accepted but is not sorted.\n", member.MemberName)
			unknownUsers = append(unknownUsers, member.MemberName)
		}
	}

	if len(unknownUsers) > 0 {
		fmt.Printf("Unknown users requesting invites: %s\n", strings.Join(unknownUsers, ", "))
	}

	if len(membersToAdd) == 0 {
		fmt.Println("No new members to add.")
		return nil
	}

	err = c.acceptPrivateLeagueInvites(membersToAdd)
	if err != nil {
		return err
	}

	fmt.Printf("%d Invites accepted successfully.\n", len(membersToAdd))
	return nil
}

func HandlePrivateLeagueInvites(bplToken, poeSessID string) error {
	client, err := NewClient(poeSessID, bplToken)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return err
	}
	return client.HandlePrivateLeagueInvites()
}

func RunContinuous(bplToken, poeSessID string, interval time.Duration) {
	client, err := NewClient(poeSessID, bplToken)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}
	for {
		fmt.Printf("%s Checking for guild invites...\n", time.Now().Format("2006-01-02 15:04:05"))
		if err := client.HandlePrivateLeagueInvites(); err != nil {
			fmt.Printf("Error handling guild invites: %v\n", err)
		}
		time.Sleep(interval)
	}
}
