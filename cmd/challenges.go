package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"fastlol/api"
	"fastlol/internal"
	"fastlol/internal/i18n"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var challengesCmd = &cobra.Command{
	Use:   "challenges [name] [tag]",
	Short: "View challenge and achievement stats",
	Long:  "View a summoner's challenge and achievement stats.",
	Args: cobra.RangeArgs(1, 2),
	Run:  runChallenges,
}

func init() {
	challengesCmd.Flags().StringP("region", "r", "", "Server region (e.g. kr, na1, euw1)")
	challengesCmd.Flags().IntP("limit", "n", 5, "Number of challenges to show")
	rootCmd.AddCommand(challengesCmd)
}

func runChallenges(cmd *cobra.Command, args []string) {
	key := viper.GetString("riot_api_key")
	if key == "" {
		internal.Error(i18n.T("error.no_riot_key"))
		fmt.Fprintln(os.Stderr, fmt.Sprintf(i18n.T("error.set_key_hint"), "riot_api_key"))
		os.Exit(1)
	}

	region, _ := cmd.Flags().GetString("region")
	if region == "" {
		region = viper.GetString("default_region")
	}
	if region == "" {
		region = "kr"
	}
	region = strings.ToLower(region)

	limit, _ := cmd.Flags().GetInt("limit")

	gameName := args[0]
	tagLine := ""
	if len(args) >= 2 {
		tagLine = args[1]
	}
	if tagLine == "" && strings.Contains(gameName, "#") {
		parts := strings.SplitN(gameName, "#", 2)
		gameName = parts[0]
		tagLine = parts[1]
	}

	displayName := gameName
	if tagLine != "" {
		displayName = fmt.Sprintf("%s#%s", gameName, tagLine)
	}

	client := api.NewRiotClient(key)
	internal.Title(fmt.Sprintf(i18n.T("challenges.title"), displayName, region))

	account, err := client.GetAccountByTag(region, gameName, tagLine)
	if err != nil {
		internal.Error(fmt.Sprintf(i18n.T("error.not_found"), err))
		os.Exit(1)
	}

	challenges, err := client.GetPlayerChallenges(region, account.PUUID)
	if err != nil {
		internal.Warn(fmt.Sprintf(i18n.T("error.fetch_failed"), err))
		fmt.Println("\033[90m  " + i18n.T("challenges.privacy_hint") + "\033[0m")
		return
	}

	if len(challenges) == 0 {
		fmt.Println("  " + i18n.T("challenges.no_data"))
		return
	}

	// Sort by level then percentile
	sort.Slice(challenges, func(i, j int) bool {
		li := levelRank(challenges[i].Level)
		lj := levelRank(challenges[j].Level)
		if li != lj {
			return li > lj
		}
		return challenges[i].Percentile > challenges[j].Percentile
	})

	if limit > 0 && limit < len(challenges) {
		challenges = challenges[:limit]
	}

	headersStr := i18n.T("challenges.headers")
	headers := strings.Split(headersStr, ",")
	var rows [][]string

	for i, c := range challenges {
		levelEmoji := levelEmojiStr(c.Level)
		percentile := fmt.Sprintf("%.1f%%", c.Percentile*100)
		value := formatLargeNumber(c.CurrentValue)

		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			fmt.Sprintf("%s %s", levelEmoji, c.Level),
			percentile,
			value,
		})
	}

	internal.Table(headers, rows)
}

func levelRank(level string) int {
	switch level {
	case "CHALLENGER": return 10
	case "GRANDMASTER": return 9
	case "MASTER": return 8
	case "DIAMOND": return 7
	case "EMERALD": return 6
	case "PLATINUM": return 5
	case "GOLD": return 4
	case "SILVER": return 3
	case "BRONZE": return 2
	case "IRON": return 1
	default: return 0
	}
}

func levelEmojiStr(level string) string {
	switch level {
	case "CHALLENGER": return "👑"
	case "GRANDMASTER": return "🥇"
	case "MASTER": return "🥈"
	case "DIAMOND": return "💎"
	case "EMERALD": return "💚"
	case "PLATINUM": return "🟪"
	case "GOLD": return "🥉"
	case "SILVER": return "🥛"
	case "BRONZE": return "🟤"
	case "IRON": return "⚫"
	default: return "❓"
	}
}

func formatLargeNumber(n int64) string {
	if n >= 1000000000 {
		return fmt.Sprintf("%.1fB", float64(n)/1000000000)
	}
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
