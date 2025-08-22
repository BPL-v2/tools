package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"tools/checkplayernames"
	"tools/guildstashtool"
	"tools/handleprivateleagueinvites"

	"github.com/AlecAivazis/survey/v2"
	"github.com/joho/godotenv"
)

var (
	bplBaseUrl      string
	bplToken        string
	poeSessID       string
	guildId         string
	privateLeagueId string
)

func init() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	}

	bplBaseUrl = "https://v2202503259898322516.goodsrv.de/api"
	bplToken = os.Getenv("BPL_TOKEN")
	poeSessID = os.Getenv("POESESSID")
	guildId = os.Getenv("GUILD_ID")
	privateLeagueId = os.Getenv("PRIVATE_LEAGUE_ID")
}

type MenuOption struct {
	Name        string
	Description string
	Action      func() error
}

func getMainMenuOptions() []MenuOption {
	return []MenuOption{
		{
			Name:        "Check Player Names",
			Description: "Check if player names contain their team abbreviations",
			Action:      showCheckPlayerNamesMenu,
		},
		{
			Name:        "Handle Private League Invites",
			Description: "Process and accept private league invites for registered players",
			Action:      showPrivateLeagueInvitesMenu,
		},
		{
			Name:        "Guild Stash Monitor",
			Description: "Monitor guild stash changes and sync to BPL backend",
			Action:      showGuildStashMenu,
		},
		{
			Name:        "Exit",
			Description: "Exit the application",
			Action: func() error {
				fmt.Println("Goodbye!")
				os.Exit(0)
				return nil
			},
		},
	}
}

func runCheckPlayerNamesSingle() error {
	if bplBaseUrl == "" {
		return fmt.Errorf("BPL_BASE_URL environment variable is required")
	}

	fmt.Println("Running player name check...")
	return checkplayernames.TeamCheck(bplBaseUrl)
}

func runCheckPlayerNamesContinuous() error {
	if bplBaseUrl == "" {
		return fmt.Errorf("BPL_BASE_URL environment variable is required")
	}

	fmt.Println("Starting continuous player name monitoring (every 5 minutes)...")
	fmt.Println("Press Ctrl+C to stop")
	checkplayernames.RunContinuous(bplBaseUrl, 5*time.Minute)
	return nil
}

func runPrivateLeagueInvitesSingle() error {
	if bplToken == "" {
		return fmt.Errorf("BPL_TOKEN environment variable is required")
	}
	if poeSessID == "" {
		return fmt.Errorf("POESESSID environment variable is required")
	}
	if privateLeagueId == "" {
		return fmt.Errorf("PRIVATE_LEAGUE_ID environment variable is required")
	}

	fmt.Println("Processing private league invites...")
	return handleprivateleagueinvites.HandlePrivateLeagueInvites(bplBaseUrl, bplToken, poeSessID, privateLeagueId)
}

func runPrivateLeagueInvitesContinuous() error {
	if bplToken == "" {
		return fmt.Errorf("BPL_TOKEN environment variable is required")
	}
	if poeSessID == "" {
		return fmt.Errorf("POESESSID environment variable is required")
	}
	if privateLeagueId == "" {
		return fmt.Errorf("PRIVATE_LEAGUE_ID environment variable is required")
	}

	fmt.Println("Starting continuous private league invite monitoring (every 5 minutes)...")
	fmt.Println("Press Ctrl+C to stop")
	handleprivateleagueinvites.RunContinuous(bplBaseUrl, bplToken, poeSessID, privateLeagueId, 5*time.Minute)
	return nil
}

func runGuildStashSingle() error {
	if poeSessID == "" {
		return fmt.Errorf("POESESSID environment variable is required")
	}
	if bplToken == "" {
		return fmt.Errorf("BPL_TOKEN environment variable is required")
	}
	if guildId == "" {
		return fmt.Errorf("GUILD_ID environment variable is required")
	}

	fmt.Println("Running guild stash monitoring...")
	return guildstashtool.RunStashMonitoring(poeSessID, bplToken, guildId)
}

func runGuildStashContinuous() error {
	if poeSessID == "" {
		return fmt.Errorf("POESESSID environment variable is required")
	}
	if bplToken == "" {
		return fmt.Errorf("BPL_TOKEN environment variable is required")
	}
	if guildId == "" {
		return fmt.Errorf("GUILD_ID environment variable is required")
	}

	fmt.Println("Starting continuous guild stash monitoring (every 5 minutes)...")
	fmt.Println("Press Ctrl+C to stop")
	return guildstashtool.RunStashMonitoringContinuous(poeSessID, bplToken, guildId, 5*time.Minute)
}

func showRunModeMenu(toolName string, singleAction, continuousAction func() error) error {
	options := []MenuOption{
		{
			Name:        "Run Once",
			Description: fmt.Sprintf("Run %s once", toolName),
			Action:      singleAction,
		},
		{
			Name:        "Run Continuously",
			Description: fmt.Sprintf("Run %s continuously every 5 minutes", toolName),
			Action:      continuousAction,
		},
		{
			Name:        "Back to Main Menu",
			Description: "Return to the main menu",
			Action: func() error {
				return nil // Just return without error to go back to main menu
			},
		},
	}

	optionNames := make([]string, len(options))
	for i, option := range options {
		optionNames[i] = option.Name
	}

	var selected string
	prompt := &survey.Select{
		Message: fmt.Sprintf("%s - Select run mode:", toolName),
		Options: optionNames,
		Description: func(value string, index int) string {
			if index < len(options) {
				return options[index].Description
			}
			return ""
		},
	}

	err := survey.AskOne(prompt, &selected)
	if err != nil {
		return err
	}

	// Find the selected option and execute its action
	for _, option := range options {
		if option.Name == selected {
			return option.Action()
		}
	}

	return fmt.Errorf("unknown option selected")
}

func showCheckPlayerNamesMenu() error {
	return showRunModeMenu("Check Player Names", runCheckPlayerNamesSingle, runCheckPlayerNamesContinuous)
}

func showPrivateLeagueInvitesMenu() error {
	return showRunModeMenu("Handle Private League Invites", runPrivateLeagueInvitesSingle, runPrivateLeagueInvitesContinuous)
}

func showGuildStashMenu() error {
	return showRunModeMenu("Guild Stash Monitor", runGuildStashSingle, runGuildStashContinuous)
}

func showMainMenu() error {
	options := getMainMenuOptions()
	optionNames := make([]string, len(options))

	for i, option := range options {
		optionNames[i] = option.Name
	}

	var selected string
	prompt := &survey.Select{
		Message: "BPL Tools - Select an option:",
		Options: optionNames,
		Description: func(value string, index int) string {
			if index < len(options) {
				return options[index].Description
			}
			return ""
		},
	}

	err := survey.AskOne(prompt, &selected)
	if err != nil {
		return err
	}

	// Find the selected option and execute its action
	for _, option := range options {
		if option.Name == selected {
			return option.Action()
		}
	}

	return fmt.Errorf("unknown option selected")
}

func main() {
	fmt.Println("Welcome to BPL Tools!")
	fmt.Println("===================")

	for {
		err := showMainMenu()
		if err != nil {
			log.Printf("Error: %v", err)

			// Ask if user wants to continue or exit
			var continueChoice bool
			continuePrompt := &survey.Confirm{
				Message: "An error occurred. Do you want to continue?",
				Default: true,
			}

			if surveyErr := survey.AskOne(continuePrompt, &continueChoice); surveyErr != nil {
				log.Fatalf("Error: %v", surveyErr)
			}

			if !continueChoice {
				fmt.Println("Goodbye!")
				break
			}
		}
	}
}
