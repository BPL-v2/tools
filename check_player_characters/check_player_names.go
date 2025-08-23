package check_player_characters

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
)

type Character struct {
	Ascendancy string `json:"ascendancy"`
}

type LadderEntry struct {
	UserID        int       `json:"user_id"`
	CharacterName string    `json:"character_name"`
	Level         int       `json:"level"`
	Character     Character `json:"character"`
}

type User struct {
	ID int `json:"id"`
}

type Team struct {
	ID             int      `json:"id"`
	Name           string   `json:"name"`
	AllowedClasses []string `json:"allowed_classes"`
}

var baseClasses = []string{
	"Scion",
	"Marauder",
	"Ranger",
	"Shadow",
	"Templar",
	"Witch",
	"Duelist",
}
var bplBaseUrl = "https://v2202503259898322516.goodsrv.de/api"

func getTeams() ([]Team, error) {
	resp, err := http.Get(bplBaseUrl + "/events/current/teams")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var teams []Team
	err = json.Unmarshal(body, &teams)
	if err != nil {
		return nil, err
	}
	return teams, nil
}

func getLadder() ([]LadderEntry, error) {
	resp, err := http.Get(bplBaseUrl + "/events/current/ladder")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var ladder []LadderEntry
	err = json.Unmarshal(body, &ladder)
	return ladder, err
}

func getUsers() (map[int]int, error) {
	resp, err := http.Get(bplBaseUrl + "/events/current/users")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var teamUsers map[string][]User
	err = json.Unmarshal(body, &teamUsers)
	if err != nil {
		return nil, err
	}

	userMap := make(map[int]int)
	for teamIDStr, users := range teamUsers {
		var teamID int
		fmt.Sscanf(teamIDStr, "%d", &teamID)
		for _, user := range users {
			userMap[user.ID] = teamID
		}
	}

	return userMap, nil
}

func CharacterCheck() error {
	userMap, err := getUsers()
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	teams, err := getTeams()
	if err != nil {
		return fmt.Errorf("failed to get teams: %w", err)
	}
	teamMap := make(map[int]Team)
	for _, team := range teams {
		teamMap[team.ID] = team
	}

	ladderEntries, err := getLadder()
	if err != nil {
		return fmt.Errorf("failed to get ladder: %w", err)
	}
	foundMismatch := false

	for _, entry := range ladderEntries {
		if teamID, exists := userMap[entry.UserID]; exists {
			if team, teamExists := teamMap[teamID]; teamExists {

				teamShort := strings.ToLower(team.Name[:3])
				if !strings.Contains(strings.ToLower(entry.CharacterName), teamShort) && entry.Level >= 10 {
					foundMismatch = true
					fmt.Printf("Mismatch: Lvl %d %s does not contain %s abbrevation\n", entry.Level, entry.CharacterName, team.Name)
				}
				if !slices.Contains(baseClasses, entry.Character.Ascendancy) &&
					!slices.Contains(team.AllowedClasses, entry.Character.Ascendancy) {
					fmt.Printf("Mismatch: %s has an invalid ascendancy: %s\n", entry.CharacterName, entry.Character.Ascendancy)
				}
			}
		}
	}
	if !foundMismatch {
		fmt.Println("No mismatches found.")
	}

	return nil
}

func RunContinuous(interval time.Duration) {
	for {
		fmt.Printf("%s Checking for player name mismatches...\n", time.Now().Format("2006-01-02 15:04:05"))
		if err := CharacterCheck(); err != nil {
			fmt.Printf("Error checking player names: %v\n", err)
		}
		time.Sleep(interval)
	}
}
