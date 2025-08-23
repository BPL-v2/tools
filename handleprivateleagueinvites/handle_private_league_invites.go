package handleprivateleagueinvites

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CredentialError represents an error related to invalid credentials
type CredentialError struct {
	Type    string // "poe_session", "bpl_token", "private_league_id"
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
	Value int    `json:"value"`
}

func getLeagueJoinRequests(poeSessID string, privateLeagueId string) ([]Member, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://www.pathofexile.com/api/private-league-member/%s", privateLeagueId), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("user-agent", "Liberatorist@gmail.com")
	req.AddCookie(&http.Cookie{Name: "POESESSID", Value: poeSessID})

	q := req.URL.Query()
	q.Add("sort", "roleDesc")
	q.Add("search", "")
	q.Add("offset", "0")
	q.Add("limit", "100")
	q.Add("_", fmt.Sprintf("%d", time.Now().Unix()))
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, NewCredentialError("poe_session", fmt.Sprintf("HttpStatusCode: %d (PoE Session ID invalid)", resp.StatusCode), resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
		return nil, NewCredentialError("private_league_id", fmt.Sprintf("HttpStatusCode: %d (Private League ID invalid)", resp.StatusCode), resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HttpStatusCode: %d from PoE API", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var membersResp MembersResponse
	err = json.Unmarshal(body, &membersResp)
	if err != nil {
		return nil, err
	}

	var requestedMembers []Member
	for _, member := range membersResp.Members {
		if member.Role == "requested_invite" {
			requestedMembers = append(requestedMembers, member)
		}
	}

	return requestedMembers, nil
}

func getSortedUsers(baseURL, bplToken string) (map[string]bool, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", baseURL+"/events/current/signups", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+bplToken)

	resp, err := client.Do(req)
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

func acceptPrivateLeagueInvites(members []Member, poeSessID string, privateLeagueId string) error {
	client := &http.Client{}

	var acceptRequests []AcceptRequest
	for _, member := range members {
		acceptRequests = append(acceptRequests, AcceptRequest{
			Name:  "accept",
			Value: member.ID,
		})
	}

	jsonData, err := json.Marshal(acceptRequests)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://www.pathofexile.com/api/private-league-member/%s", privateLeagueId), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("user-agent", "Contact: Liberatorist@gmail.com")
	req.AddCookie(&http.Cookie{Name: "POESESSID", Value: poeSessID})

	resp, err := client.Do(req)
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

func HandlePrivateLeagueInvites(baseURL string, bplToken string, poeSessID string, privateLeagueId string) error {
	sortedUsers, err := getSortedUsers(baseURL, bplToken)
	if err != nil {
		return fmt.Errorf("failed to get sorted users: %w", err)
	}

	guildRequests, err := getLeagueJoinRequests(poeSessID, privateLeagueId)
	if err != nil {
		return fmt.Errorf("failed to get guild join requests: %w", err)
	}

	var membersToAdd []Member
	var unknownUsers []string

	for _, member := range guildRequests {
		if sortedUsers[member.MemberName] && member.IsAcceptable {
			membersToAdd = append(membersToAdd, member)
		} else if !sortedUsers[member.MemberName] {
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

	err = acceptPrivateLeagueInvites(membersToAdd, poeSessID, privateLeagueId)
	if err != nil {
		return err
	}

	fmt.Printf("%d Invites accepted successfully.\n", len(membersToAdd))
	return nil
}

func RunContinuous(baseURL, bplToken, poeSessID string, privateLeagueId string, interval time.Duration) {
	for {
		fmt.Printf("%s Checking for guild invites...\n", time.Now().Format("2006-01-02 15:04:05"))
		if err := HandlePrivateLeagueInvites(baseURL, bplToken, poeSessID, privateLeagueId); err != nil {
			fmt.Printf("Error handling guild invites: %v\n", err)
		}
		time.Sleep(interval)
	}
}
