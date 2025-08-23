package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"tools/check_player_characters"
	"tools/guild_stash_logs"
	"tools/league_invites"

	"github.com/AlecAivazis/survey/v2"
)

var (
	bplBaseUrl      string
	bplToken        string
	poeSessID       string
	guildId         string
	privateLeagueId string
)

func init() {
	// Load environment variables from bpl-config.txt file
	loadEnvFromFile("bpl-config.txt")

	bplBaseUrl = "https://v2202503259898322516.goodsrv.de/api"
	bplToken = os.Getenv("BPL_TOKEN")
	poeSessID = os.Getenv("POESESSID")
	guildId = os.Getenv("GUILD_ID")
	privateLeagueId = os.Getenv("PRIVATE_LEAGUE_ID")
}

// loadEnvFromFile loads environment variables from a file
func loadEnvFromFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		// File doesn't exist yet, which is fine
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			os.Setenv(key, value)
		}
	}
}

// EnvVar represents an environment variable with its description
type EnvVar struct {
	Name        string
	Description string
	Required    bool
}

// getEnvVarInstructions returns instructions for how to obtain specific environment variables
func getEnvVarInstructions(envVarName string) string {
	switch envVarName {
	case "BPL_TOKEN":
		return `
═══════════════════════════════════════════════════════════════
                    How to get your BPL Token
═══════════════════════════════════════════════════════════════

1. Navigate to https://bpl-poe.com/
2. Make sure you are logged in
3. Press F12 to open dev tools
4. Go to Application (Chromium) / Storage (Firefox)
5. Go to Local storage for https://bpl-poe.com/
6. Select value for 'auth'
7. Copy the entire token value

═══════════════════════════════════════════════════════════════`

	case "POESESSID":
		return `
═══════════════════════════════════════════════════════════════
                 How to get your PoE Session ID
═══════════════════════════════════════════════════════════════

⚠️  IMPORTANT: This is EXTREMELY sensitive data! 
    Please check the code to see how it's used.

1. Navigate to https://www.pathofexile.com/
2. Make sure you are logged in
3. Press F12 to open dev tools
4. Go to Application (Chromium) / Storage (Firefox)
5. Go to Cookies for https://www.pathofexile.com/
6. Find the POESESSID cookie and copy its value

═══════════════════════════════════════════════════════════════`

	case "GUILD_ID":
		return `
═══════════════════════════════════════════════════════════════
                    How to get your Guild ID
═══════════════════════════════════════════════════════════════

1. Navigate to https://www.pathofexile.com/my-guild
2. Click on "Stash History"
3. Your Browser URL should look like: 
   https://www.pathofexile.com/guild/profile/408208/stash-history
4. The number between 'profile' and 'stash-history' is your guild ID
   (e.g., 408208 in the example above)

═══════════════════════════════════════════════════════════════`

	case "PRIVATE_LEAGUE_ID":
		return `
═══════════════════════════════════════════════════════════════
                How to get your Private League ID
═══════════════════════════════════════════════════════════════

1. Navigate to https://www.pathofexile.com/private-leagues
2. Click "view" on your league
3. It will be named "YourLeagueName (PL12345)"
4. The number after "PL" is your league ID
   (e.g., 12345 in the example above)

═══════════════════════════════════════════════════════════════`

	default:
		return ""
	}
}

// promptForEnvVarWithoutInstructions prompts the user to enter a value without showing instructions
func promptForEnvVarWithoutInstructions(envVar EnvVar) (string, error) {
	var value string
	var prompt survey.Prompt

	if strings.Contains(strings.ToUpper(envVar.Name), "TOKEN") || strings.Contains(strings.ToUpper(envVar.Name), "SESS") {
		prompt = &survey.Password{
			Message: fmt.Sprintf("Enter %s (%s):", envVar.Name, envVar.Description),
		}
	} else {
		prompt = &survey.Input{
			Message: fmt.Sprintf("Enter %s (%s):", envVar.Name, envVar.Description),
		}
	}

	err := survey.AskOne(prompt, &value, survey.WithValidator(survey.Required))
	return value, err
}

// updateEnvFile updates or creates the bpl-config.txt file with the new environment variable
func updateEnvFile(key, value string) error {
	envFile := "bpl-config.txt"

	// Read existing content
	var lines []string
	if file, err := os.Open(envFile); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
	}

	// Check if the key already exists and update it
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, key+"=") {
			lines[i] = fmt.Sprintf("%s=%s", key, value)
			found = true
			break
		}
	}

	// If not found, add it
	if !found {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	// Write back to file
	file, err := os.Create(envFile)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, line := range lines {
		if _, err := file.WriteString(line + "\n"); err != nil {
			return err
		}
	}

	return nil
}

// ensureEnvVars ensures that all required environment variables are set
func ensureEnvVars(envVars []EnvVar) error {
	for _, envVar := range envVars {
		value := os.Getenv(envVar.Name)
		if value == "" && envVar.Required {
			fmt.Printf("\n%s is required but not set.\n", envVar.Name)

			// Ask if user wants to see instructions
			var showInstructions bool
			instructionsPrompt := &survey.Confirm{
				Message: fmt.Sprintf("Do you want to see instructions on how to get %s?", envVar.Name),
				Default: true,
			}

			if err := survey.AskOne(instructionsPrompt, &showInstructions); err != nil {
				return err
			}

			if showInstructions {
				instructions := getEnvVarInstructions(envVar.Name)
				if instructions != "" {
					fmt.Println(instructions)
					fmt.Println("\nPress Enter to continue...")
					fmt.Scanln() // Wait for user to press Enter
					fmt.Println()
				}
			}

			newValue, err := promptForEnvVarWithoutInstructions(envVar)
			if err != nil {
				return err
			}

			// Set the environment variable for this session
			os.Setenv(envVar.Name, newValue)

			// Update global variables
			switch envVar.Name {
			case "BPL_TOKEN":
				bplToken = newValue
			case "POESESSID":
				poeSessID = newValue
			case "GUILD_ID":
				guildId = newValue
			case "PRIVATE_LEAGUE_ID":
				privateLeagueId = newValue
			}

			// Update the bpl-config.txt file
			if err := updateEnvFile(envVar.Name, newValue); err != nil {
				log.Printf("Warning: Could not update bpl-config.txt file: %v", err)
			} else {
				fmt.Printf("✓ %s saved to bpl-config.txt file\n", envVar.Name)
			}
		}
	}
	return nil
}

// isCredentialError checks if an error is a credential-related error
func isCredentialError(err error) (credType string, isCredError bool) {
	// Unwrap the error to check for wrapped credential errors
	var guildCredErr *guild_stash_logs.CredentialError
	if errors.As(err, &guildCredErr) {
		return guildCredErr.Type, true
	}

	var privateCredErr *league_invites.CredentialError
	if errors.As(err, &privateCredErr) {
		return privateCredErr.Type, true
	}

	return "", false
} // handleCredentialError prompts user to re-enter credentials when they're detected as faulty
func handleCredentialError(credType string) error {
	fmt.Printf("\n🚨 Credential Error Detected!\n")
	fmt.Printf("The %s appears to be invalid or expired.\n\n", getCredentialDisplayName(credType))

	var retry bool
	retryPrompt := &survey.Confirm{
		Message: "Would you like to enter a new value?",
		Default: true,
	}

	if err := survey.AskOne(retryPrompt, &retry); err != nil {
		return err
	}

	if !retry {
		return fmt.Errorf("user chose not to update credentials")
	}

	// Create EnvVar for the specific credential type
	var envVar EnvVar
	switch credType {
	case "poe_session":
		envVar = EnvVar{Name: "POESESSID", Description: "Path of Exile session ID from browser", Required: true}
	case "bpl_token":
		envVar = EnvVar{Name: "BPL_TOKEN", Description: "BPL API token for authentication", Required: true}
	case "guild_id":
		envVar = EnvVar{Name: "GUILD_ID", Description: "Guild ID to monitor stash for", Required: true}
	case "private_league_id":
		envVar = EnvVar{Name: "PRIVATE_LEAGUE_ID", Description: "Private league ID to process invites for", Required: true}
	default:
		return fmt.Errorf("unknown credential type: %s", credType)
	}

	// Ask if user wants to see instructions
	var showInstructions bool
	instructionsPrompt := &survey.Confirm{
		Message: fmt.Sprintf("Do you want to see instructions on how to get %s?", envVar.Name),
		Default: true,
	}

	if err := survey.AskOne(instructionsPrompt, &showInstructions); err != nil {
		return err
	}

	if showInstructions {
		instructions := getEnvVarInstructions(envVar.Name)
		if instructions != "" {
			fmt.Println(instructions)
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()
			fmt.Println()
		}
	}

	newValue, err := promptForEnvVarWithoutInstructions(envVar)
	if err != nil {
		return err
	}

	// Set the environment variable for this session
	os.Setenv(envVar.Name, newValue)

	// Update global variables
	switch envVar.Name {
	case "BPL_TOKEN":
		bplToken = newValue
	case "POESESSID":
		poeSessID = newValue
	case "GUILD_ID":
		guildId = newValue
	case "PRIVATE_LEAGUE_ID":
		privateLeagueId = newValue
	}

	// Update the config file
	if err := updateEnvFile(envVar.Name, newValue); err != nil {
		log.Printf("Warning: Could not update bpl-config.txt file: %v", err)
	} else {
		fmt.Printf("✓ %s updated and saved to bpl-config.txt file\n\n", envVar.Name)
	}

	return nil
}

// getCredentialDisplayName returns a user-friendly name for credential types
func getCredentialDisplayName(credType string) string {
	switch credType {
	case "poe_session":
		return "PoE Session ID"
	case "bpl_token":
		return "BPL Token"
	case "guild_id":
		return "Guild ID"
	case "private_league_id":
		return "Private League ID"
	default:
		return credType
	}
}

// runWithCredentialRetry runs a function and handles credential errors with retry logic
func runWithCredentialRetry(fn func() error) error {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := fn()
		if err == nil {
			return nil
		}

		credType, isCredError := isCredentialError(err)
		if !isCredError {
			return err // Not a credential error, return original error
		}

		fmt.Printf("Attempt %d/%d failed: %v\n", i+1, maxRetries, err)

		if i == maxRetries-1 {
			return fmt.Errorf("maximum retries reached. Last error: %w", err)
		}

		if err := handleCredentialError(credType); err != nil {
			return fmt.Errorf("failed to update credentials: %w", err)
		}

		fmt.Println("Retrying with updated credentials...")
	}

	return fmt.Errorf("unexpected error in retry logic")
}

type MenuOption struct {
	Name        string
	Description string
	Action      func() error
}

func getMainMenuOptions() []MenuOption {
	return []MenuOption{
		{
			Name:        "Check Player Characters",
			Description: "Check if characters are named correctly and have allowed ascendancies",
			Action:      showCheckCharactersMenu,
		},
		{
			Name:        "Handle Private League Invites",
			Description: "Process and accept private league invites for registered players",
			Action:      showPrivateLeagueInvitesMenu,
		},
		{
			Name:        "Guild Stash Logs",
			Description: "Fetch guild stash changes and sync to BPL backend",
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
	envVars := []EnvVar{
		{Name: "BPL_TOKEN", Description: "BPL API token for authentication", Required: false},
	}

	if err := ensureEnvVars(envVars); err != nil {
		return err
	}

	fmt.Println("Running player name check...")
	return check_player_characters.CharacterCheck(bplBaseUrl)
}

func runCheckPlayerNamesContinuous() error {
	envVars := []EnvVar{
		{Name: "BPL_TOKEN", Description: "BPL API token for authentication", Required: false},
	}

	if err := ensureEnvVars(envVars); err != nil {
		return err
	}

	fmt.Println("Starting continuous player name monitoring (every 5 minutes)...")
	fmt.Println("Press Ctrl+C to stop")
	check_player_characters.RunContinuous(bplBaseUrl, 5*time.Minute)
	return nil
}

func runPrivateLeagueInvitesSingle() error {
	envVars := []EnvVar{
		{Name: "BPL_TOKEN", Description: "BPL API token for authentication", Required: true},
		{Name: "POESESSID", Description: "Path of Exile session ID from browser", Required: true},
		{Name: "PRIVATE_LEAGUE_ID", Description: "Private league ID to process invites for", Required: true},
	}

	if err := ensureEnvVars(envVars); err != nil {
		return err
	}

	fmt.Println("Processing private league invites...")
	return runWithCredentialRetry(func() error {
		return league_invites.HandlePrivateLeagueInvites(bplBaseUrl, bplToken, poeSessID, privateLeagueId)
	})
}

func runPrivateLeagueInvitesContinuous() error {
	envVars := []EnvVar{
		{Name: "BPL_TOKEN", Description: "BPL API token for authentication", Required: true},
		{Name: "POESESSID", Description: "Path of Exile session ID from browser", Required: true},
		{Name: "PRIVATE_LEAGUE_ID", Description: "Private league ID to process invites for", Required: true},
	}

	if err := ensureEnvVars(envVars); err != nil {
		return err
	}

	fmt.Println("Starting continuous private league invite monitoring (every 5 minutes)...")
	fmt.Println("Press Ctrl+C to stop")
	return runWithCredentialRetry(func() error {
		league_invites.RunContinuous(bplBaseUrl, bplToken, poeSessID, privateLeagueId, 5*time.Minute)
		return nil
	})
}

func runGuildStashSingle() error {
	envVars := []EnvVar{
		{Name: "POESESSID", Description: "Path of Exile session ID from browser", Required: true},
		{Name: "BPL_TOKEN", Description: "BPL API token for authentication", Required: true},
		{Name: "GUILD_ID", Description: "Guild ID to monitor stash for", Required: true},
	}

	if err := ensureEnvVars(envVars); err != nil {
		return err
	}

	fmt.Println("Running guild stash monitoring...")
	return runWithCredentialRetry(func() error {
		return guild_stash_logs.RunStashMonitoring(poeSessID, bplToken, guildId)
	})
}

func runGuildStashContinuous() error {
	envVars := []EnvVar{
		{Name: "POESESSID", Description: "Path of Exile session ID from browser", Required: true},
		{Name: "BPL_TOKEN", Description: "BPL API token for authentication", Required: true},
		{Name: "GUILD_ID", Description: "Guild ID to monitor stash for", Required: true},
	}

	if err := ensureEnvVars(envVars); err != nil {
		return err
	}

	fmt.Println("Starting continuous guild stash monitoring (every 5 minutes)...")
	fmt.Println("Press Ctrl+C to stop")
	return runWithCredentialRetry(func() error {
		return guild_stash_logs.RunStashMonitoringContinuous(poeSessID, bplToken, guildId, 5*time.Minute)
	})
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

func showCheckCharactersMenu() error {
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
