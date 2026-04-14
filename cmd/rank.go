package cmd

import (
	"fmt"
	"os"
	"strings"

	"fastlol/api"
	"fastlol/internal"
	"fastlol/internal/i18n"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rankCmd = &cobra.Command{
	Use:   "rank [name] [tag]",
	Short: "Check summoner's ranked tier and LP",
	Long:  "Check a summoner's current ranked tier, LP, wins, losses, and win rate.",
	Args:  cobra.RangeArgs(1, 2),
	Run:   runRank,
}

func init() {
	rankCmd.Flags().StringP("region", "r", "", "Server region (e.g. kr, na1, euw1)")
	rootCmd.AddCommand(rankCmd)
}

func runRank(cmd *cobra.Command, args []string) {
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
	internal.Title(fmt.Sprintf(i18n.T("rank.title"), displayName, region))

	account, err := client.GetAccountByTag(region, gameName, tagLine)
	if err != nil {
		internal.Error(fmt.Sprintf(i18n.T("error.not_found"), err))
		os.Exit(1)
	}

	fmt.Printf("\n  \033[1;32m%s#%s\033[0m\n\n", account.GameName, account.TagLine)

	entries, err := client.GetLeagueEntriesByPUUID(region, account.PUUID)
	if err != nil {
		internal.Warn(fmt.Sprintf("无法获取排位数据: %v", err))
		fmt.Println("\033[90m  " + i18n.T("rank.privacy_warn") + "\033[0m")
		return
	}

	if len(entries) == 0 {
		fmt.Println("  " + i18n.T("rank.no_data"))
		return
	}

	displayRankedEntries(entries)
}

func displayRankedEntries(entries []api.LeagueEntry) {
	queueNames := map[string]string{
		"RANKED_SOLO_5x5": i18n.GetLocalizedQueue("RANKED_SOLO_5x5"),
		"RANKED_FLEX_SR":   i18n.GetLocalizedQueue("RANKED_FLEX_SR"),
		"RANKED_TFT":       i18n.GetLocalizedQueue("RANKED_TFT"),
	}

	tierColors := map[string]string{
		"IRON":     "\033[90m",
		"BRONZE":   "\033[33m",
		"SILVER":   "\033[37m",
		"GOLD":     "\033[33m",
		"PLATINUM": "\033[32m",
		"EMERALD":  "\033[32m",
		"DIAMOND":  "\033[36m",
		"MASTER":   "\033[35m",
		"GRANDMASTER": "\033[31m",
		"CHALLENGER": "\033[33m",
	}

	for _, e := range entries {
		qName := queueNames[e.QueueType]
		if qName == "" {
			qName = e.QueueType
		}

		color := tierColors[e.Tier]
		if color == "" {
			color = "\033[37m"
		}

		winRate := api.WinRate(e.Wins, e.Losses)
		hotStreak := ""
		if e.HotStreak {
			hotStreak = " 🔥"
		}

		tierDisplay := fmt.Sprintf("%s%s %s\033[0m", color, e.Tier, e.Rank)

		fmt.Printf("  %s | %s | %d LP%s\n", qName, tierDisplay, e.LeaguePoints, hotStreak)
		fmt.Println(fmt.Sprintf("\n  %d%s %d%s | %s %.1f%%\n\n",
			e.Wins, i18n.T("profile.win"), e.Losses, i18n.T("profile.loss"),
			"WR", winRate))
	}
}
