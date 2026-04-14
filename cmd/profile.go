package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"fastlol/api"
	"fastlol/internal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var profileCmd = &cobra.Command{
	Use:   "profile <summoner> [tag]",
	Short: "Show summoner profile, champion mastery, and recent matches",
	Long: `Look up a summoner's profile using Riot API:
  - Basic info from Riot ID (via account-v1)
  - Champion mastery (top champions)
  - Recent match history (via match-v5)

Note: Development API keys have limited access to summoner-v4 and ranked data.

For Riot ID format: fastlol profile "GameName" TAG
For legacy names: fastlol profile "SummonerName"

Default region can be set in config or with --region flag.`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runProfile,
}

func init() {
	profileCmd.Flags().StringP("region", "r", "", "Server region (na1, euw1, kr, etc.)")
	profileCmd.Flags().IntP("matches", "n", 5, "Number of recent matches to show (max 20)")
	profileCmd.Flags().IntP("mastery", "m", 5, "Number of top champions to show")
	profileCmd.Flags().Bool("mock", false, "Use mock data for testing (no API key needed)")
	rootCmd.AddCommand(profileCmd)
}

func runProfile(cmd *cobra.Command, args []string) {
	useMock, _ := cmd.Flags().GetBool("mock")

	key := viper.GetString("riot_api_key")
	if !useMock && key == "" {
		internal.Error("No Riot API key configured.")
		fmt.Fprintln(os.Stderr, "Get one at https://developer.riotgames.com/")
		fmt.Fprintln(os.Stderr, "Then set it: fastlol config set riot_api_key <your-key>")
		fmt.Fprintln(os.Stderr, "\nOr use --mock flag for testing with fake data:")
		fmt.Fprintf(os.Stderr, "  fastlol profile %s --mock\n", args[0])
		os.Exit(1)
	}

	region, _ := cmd.Flags().GetString("region")
	if region == "" {
		region = viper.GetString("default_region")
	}
	if region == "" {
		region = "na1"
	}
	region = strings.ToLower(region)

	matchCount, _ := cmd.Flags().GetInt("matches")
	masteryCount, _ := cmd.Flags().GetInt("mastery")

	gameName := args[0]
	tagLine := ""
	if len(args) >= 2 {
		tagLine = args[1]
	}

	// Handle Riot ID format with #
	if tagLine == "" && strings.Contains(gameName, "#") {
		parts := strings.SplitN(gameName, "#", 2)
		gameName = parts[0]
		tagLine = parts[1]
	}

	displayName := gameName
	if tagLine != "" {
		displayName = fmt.Sprintf("%s#%s", gameName, tagLine)
	}

	if useMock {
		internal.Title(fmt.Sprintf("[MOCK] Profile: %s (%s)", displayName, region))
		runMockProfile(displayName, region, matchCount, masteryCount)
		return
	}

	client := api.NewRiotClient(key)
	internal.Title(fmt.Sprintf("Looking up: %s (%s)", displayName, region))

	// Step 1: Get account via account-v1 (always available)
	var account *api.Account
	var err error

	if tagLine != "" {
		account, err = client.GetAccountByTag(region, gameName, tagLine)
	} else {
		// Try legacy summoner name lookup
		summoner, err := client.GetSummonerByName(region, gameName)
		if err == nil {
			account = &api.Account{
				PUUID:    summoner.PUUID,
				GameName: gameName,
				TagLine:  "",
			}
		} else {
			// Fallback: try as Riot ID with empty tag or use as-is
			account = &api.Account{
				PUUID:    "",
				GameName: gameName,
				TagLine:  tagLine,
			}
			internal.Warn("Could not resolve summoner name. For best results, use 'Name#TAG' format.")
		}
	}

	if err != nil || account == nil || account.PUUID == "" {
		internal.Error(fmt.Sprintf("Failed to find account: %v", err))
		fmt.Fprintln(os.Stderr, "\nTip: Use Riot ID format: 'Name#TAG'")
		fmt.Fprintln(os.Stderr, "Example: fastlol profile 'Hide on bush' KR")
		os.Exit(1)
	}

	displayAccountInfo(account)

	// Step 2: Try to get champion mastery (available for dev keys)
	masteries, err := client.GetChampionMastery(region, account.PUUID, masteryCount)
	if err == nil && len(masteries) > 0 {
		displayChampionMastery(masteries)
	}

	// Step 3: Get recent matches (available for dev keys)
	if matchCount > 0 {
		matchIDs, err := client.GetRecentMatches(region, account.PUUID, matchCount)
		if err == nil && len(matchIDs) > 0 {
			matches := fetchMatchDetails(client, region, account.PUUID, matchIDs)
			if len(matches) > 0 {
				displayRecentMatches(matches)
			}
		} else if err != nil {
			internal.Warn(fmt.Sprintf("Could not fetch matches: %v", err))
		}
	}

	// Note about limited data
	fmt.Println()
	fmt.Println("\033[90m  Note: Development API keys have limited access.")
	fmt.Println("        Summoner level and ranked stats may be unavailable.\033[0m")
}

func displayAccountInfo(a *api.Account) {
	fmt.Printf("\033[1;32m  %s#%s\033[0m\n", a.GameName, a.TagLine)
	fmt.Printf("  PUUID: %s...%s\n", a.PUUID[:8], a.PUUID[len(a.PUUID)-8:])
	fmt.Println()
}

func displayChampionMastery(masteries []api.ChampionMastery) {
	fmt.Println("\033[1m  Champion Mastery:\033[0m")
	fmt.Println()

	headers := []string{"Rank", "Champion ID", "Level", "Points", "Last Played"}
	var rows [][]string

	for i, m := range masteries {
		lastPlayed := time.UnixMilli(m.LastPlayTime).Format("01/02")
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			fmt.Sprintf("%d", m.ChampionID),
			fmt.Sprintf("%d", m.ChampionLevel),
			fmt.Sprintf("%d", m.ChampionPoints),
			lastPlayed,
		})
	}

	internal.Table(headers, rows)
	fmt.Println()
}

func fetchMatchDetails(client *api.RiotClient, region, puuid string, matchIDs []string) []api.MatchMetadata {
	var matches []api.MatchMetadata
	for _, id := range matchIDs {
		match, err := client.GetMatchInfo(region, id)
		if err != nil {
			continue
		}
		meta := match.GetPlayerMatchMetadata(puuid)
		if meta != nil {
			matches = append(matches, *meta)
		}
	}
	return matches
}

func displayRecentMatches(matches []api.MatchMetadata) {
	fmt.Println("\033[1m  Recent Matches:\033[0m")
	fmt.Println()

	headers := []string{"Time", "Mode", "Champion", "KDA", "Result", "Duration"}
	var rows [][]string

	for _, meta := range matches {
		t := time.UnixMilli(meta.GameCreation)
		timeStr := t.Format("01/02 15:04")

		kda := fmt.Sprintf("%d/%d/%d", meta.Kills, meta.Deaths, meta.Assists)
		if meta.Deaths == 0 {
			if meta.Kills+meta.Assists > 0 {
				kda += " (Perfect)"
			}
		} else {
			ratio := float64(meta.Kills+meta.Assists) / float64(meta.Deaths)
			kda += fmt.Sprintf(" (%.2f)", ratio)
		}

		result := "\033[31mLoss\033[0m"
		if meta.Win {
			result = "\033[32mWin\033[0m"
		}

		duration := api.FormatDuration(meta.GameDuration)

		rows = append(rows, []string{
			timeStr,
			meta.GameMode,
			meta.ChampionName,
			kda,
			result,
			duration,
		})
	}

	internal.Table(headers, rows)
}

func runMockProfile(displayName, region string, matchCount, masteryCount int) {
	// Account info
	parts := strings.Split(displayName, "#")
	gameName := parts[0]
	tagLine := ""
	if len(parts) > 1 {
		tagLine = parts[1]
	}

	fmt.Printf("\033[1;32m  %s#%s\033[0m\n", gameName, tagLine)
	fmt.Printf("  PUUID: MOCK-xxxx-xxxx...\n")
	fmt.Println()

	// Mock champion mastery
	fmt.Println("\033[1m  Champion Mastery:\033[0m")
	fmt.Println()
	headers := []string{"Rank", "Champion ID", "Level", "Points", "Last Played"}
	var rows [][]string
	champions := []int{84, 126, 517, 103, 238} // Akali, Jayce, Syndra, Ahri, Zed
	for i := 0; i < masteryCount && i < len(champions); i++ {
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			fmt.Sprintf("%d", champions[i]),
			fmt.Sprintf("%d", 7-i),
			fmt.Sprintf("%d", 300000-i*50000),
			"04/14",
		})
	}
	internal.Table(headers, rows)
	fmt.Println()

	// Mock matches
	if matchCount > 0 {
		displayMockMatches(matchCount)
	}

	fmt.Println()
	fmt.Println("\033[90m  Note: Development API keys have limited access.")
	fmt.Println("        Summoner level and ranked stats may be unavailable.\033[0m")
}

func displayMockMatches(count int) {
	fmt.Println("\033[1m  Recent Matches:\033[0m")
	fmt.Println()

	headers := []string{"Time", "Mode", "Champion", "KDA", "Result", "Duration"}
	var rows [][]string

	champions := []string{"Ahri", "Yasuo", "Lee Sin", "Jinx", "Thresh", "Zed", "Viego", "Kaisa"}
	modes := []string{"CLASSIC", "ARAM", "URF"}
	now := time.Now()

	for i := 0; i < count && i < len(champions); i++ {
		t := now.Add(-time.Duration(i*2) * time.Hour)
		win := i%3 != 0
		kills := 5 + i*2
		deaths := 3 + i%4
		assists := 8 + i

		kda := fmt.Sprintf("%d/%d/%d", kills, deaths, assists)
		if deaths == 0 {
			kda += " (Perfect)"
		} else {
			ratio := float64(kills+assists) / float64(deaths)
			kda += fmt.Sprintf(" (%.2f)", ratio)
		}

		result := "\033[31mLoss\033[0m"
		if win {
			result = "\033[32mWin\033[0m"
		}

		rows = append(rows, []string{
			t.Format("01/02 15:04"),
			modes[i%len(modes)],
			champions[i],
			kda,
			result,
			fmt.Sprintf("%d:%02d", 30+i*2, 0),
		})
	}

	internal.Table(headers, rows)
}
