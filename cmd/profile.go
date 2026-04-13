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
	Short: "Show summoner profile and recent matches",
	Long: `Look up a summoner's profile including:
  - Summoner level and profile icon
  - Ranked stats for each queue
  - Recent match history (last 10 games)

For Riot ID format: fastlol profile "GameName" TAG
For legacy names: fastlol profile "SummonerName"

Default region can be set in config or with --region flag.`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runProfile,
}

func init() {
	profileCmd.Flags().StringP("region", "r", "", "Server region (na1, euw1, kr, etc.)")
	profileCmd.Flags().IntP("matches", "n", 5, "Number of recent matches to show (max 20)")
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

	gameName := args[0]
	tagLine := ""
	if len(args) >= 2 {
		tagLine = args[1]
	}

	displayName := gameName
	if tagLine != "" {
		displayName = fmt.Sprintf("%s#%s", gameName, tagLine)
	}

	var summoner *api.Summoner
	var rankedStats []api.RankedInfo
	var matches []api.MatchMetadata

	if useMock {
		internal.Title(fmt.Sprintf("[MOCK] Profile: %s (%s)", displayName, region))
		summoner, rankedStats, matches = generateMockData(displayName, region, matchCount)
	} else {
		client := api.NewRiotClient(key)

		var err error
		if tagLine != "" {
			internal.Title(fmt.Sprintf("Looking up: %s#%s (%s)", gameName, tagLine, region))
			summoner, err = client.GetSummonerByTag(region, gameName, tagLine)
		} else {
			internal.Title(fmt.Sprintf("Looking up: %s (%s)", gameName, region))
			if strings.Contains(gameName, "#") {
				parts := strings.SplitN(gameName, "#", 2)
				summoner, err = client.GetSummonerByTag(region, parts[0], parts[1])
			} else {
				summoner, err = client.GetSummonerByName(region, gameName)
			}
		}

		if err != nil {
			internal.Error(fmt.Sprintf("Failed to find summoner: %v", err))
			fmt.Fprintln(os.Stderr, "\nTip: Use 'Name#TAG' format for Riot IDs")
			os.Exit(1)
		}

		rankedStats, _ = client.GetRankedStats(region, summoner.ID)
		matchIDs, _ := client.GetRecentMatches(region, summoner.PUUID, matchCount)
		for _, id := range matchIDs {
			match, err := client.GetMatchInfo(region, id)
			if err == nil {
				meta := match.GetPlayerMatchMetadata(summoner.PUUID)
				if meta != nil {
					matches = append(matches, *meta)
				}
			}
		}
	}

	displaySummonerInfo(summoner)
	if len(rankedStats) > 0 {
		displayRankedStats(rankedStats)
	}
	if len(matches) > 0 {
		displayRecentMatches(matches)
	}
}

func generateMockData(name, region string, matchCount int) (*api.Summoner, []api.RankedInfo, []api.MatchMetadata) {
	summoner := &api.Summoner{
		ID:            "mock-id-12345",
		AccountID:     "mock-account",
		PUUID:         "mock-puuid-xxx",
		Name:          name,
		ProfileIconID: 29,
		RevisionDate:  time.Now().UnixMilli(),
		SummonerLevel: 342,
	}

	ranked := []api.RankedInfo{
		{
			QueueType:    "RANKED_SOLO_5x5",
			Tier:         "DIAMOND",
			Rank:         "IV",
			LeaguePoints: 67,
			Wins:         145,
			Losses:       132,
			HotStreak:    true,
		},
		{
			QueueType:    "RANKED_FLEX_SR",
			Tier:         "PLATINUM",
			Rank:         "II",
			LeaguePoints: 45,
			Wins:         89,
			Losses:       76,
			HotStreak:    false,
		},
	}

	champions := []string{"Ahri", "Yasuo", "Lee Sin", "Jinx", "Thresh", "Zed", "Viego", "Kaisa"}
	modes := []string{"CLASSIC", "ARAM", "URF"}

	var matches []api.MatchMetadata
	now := time.Now()
	for i := 0; i < matchCount && i < len(champions); i++ {
		win := i%3 != 0 // 2/3 win rate
		kills := 5 + i*2
		deaths := 3 + i%4
		assists := 8 + i

		matches = append(matches, api.MatchMetadata{
			MatchID:       fmt.Sprintf("MOCK_%d", i),
			GameCreation:  now.Add(-time.Duration(i*2) * time.Hour).UnixMilli(),
			GameDuration:  1800 + i*120,
			GameMode:      modes[i%len(modes)],
			GameType:      "MATCHED_GAME",
			Win:           win,
			ChampionName:  champions[i%len(champions)],
			Kills:         kills,
			Deaths:        deaths,
			Assists:       assists,
		})
	}

	return summoner, ranked, matches
}

func displaySummonerInfo(s *api.Summoner) {
	fmt.Printf("\033[1;32m  %s\033[0m\n", s.Name)
	fmt.Printf("  Level: %d\n", s.SummonerLevel)
	fmt.Printf("  Profile Icon: #%d\n", s.ProfileIconID)
	fmt.Println()
}

func displayRankedStats(stats []api.RankedInfo) {
	fmt.Println("\033[1m  Ranked Stats:\033[0m")
	fmt.Println()

	headers := []string{"Queue", "Tier", "LP", "W/L", "Win Rate", "Hot Streak"}
	var rows [][]string

	queueNames := map[string]string{
		"RANKED_SOLO_5x5": "Solo/Duo",
		"RANKED_FLEX_SR":  "Flex 5v5",
		"RANKED_FLEX_TT":  "Flex 3v3",
		"RANKED_TFT":      "TFT",
	}

	for _, s := range stats {
		queueName := queueNames[s.QueueType]
		if queueName == "" {
			queueName = s.QueueType
		}

		winRate := api.WinRate(s.Wins, s.Losses)
		wrStr := fmt.Sprintf("%.1f%%", winRate)
		if winRate >= 55 {
			wrStr = fmt.Sprintf("\033[32m%.1f%%\033[0m", winRate)
		} else if winRate < 45 {
			wrStr = fmt.Sprintf("\033[31m%.1f%%\033[0m", winRate)
		}

		hotStreak := ""
		if s.HotStreak {
			hotStreak = "🔥"
		}

		rows = append(rows, []string{
			queueName,
			fmt.Sprintf("%s %s", s.Tier, s.Rank),
			fmt.Sprintf("%d LP", s.LeaguePoints),
			fmt.Sprintf("%d/%d", s.Wins, s.Losses),
			wrStr,
			hotStreak,
		})
	}

	internal.Table(headers, rows)
	fmt.Println()
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
