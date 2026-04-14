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

var topCmd = &cobra.Command{
	Use:   "top [tier]",
	Short: "View Challenger/Grandmaster/Master leaderboard",
	Long:  "View top players in Challenger, Grandmaster, or Master tiers.",
	Args: cobra.RangeArgs(0, 1),
	Run:  runTop,
}

func init() {
	topCmd.Flags().StringP("region", "r", "", "Server region (e.g. kr, na1, euw1)")
	topCmd.Flags().IntP("limit", "n", 15, "Number of players to show")
	rootCmd.AddCommand(topCmd)
}

func runTop(cmd *cobra.Command, args []string) {
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

	tier := "challengerleagues"
	tierName := i18n.T("top.tier.challenger")

	if len(args) >= 1 {
		switch strings.ToLower(args[0]) {
		case "challenger", "ch", "王者":
			tier = "challengerleagues"
			tierName = i18n.T("top.tier.challenger")
		case "grandmaster", "gm", "大师":
			tier = "grandmasterleagues"
			tierName = i18n.T("top.tier.gm")
		case "master", "m", "宗师":
			tier = "masterleagues"
			tierName = i18n.T("top.tier.master")
		default:
			internal.Error(fmt.Sprintf(i18n.T("top.tier.unknown"), args[0]))
			fmt.Fprintln(os.Stderr, "  "+i18n.T("top.tier.valid"))
			os.Exit(1)
		}
	}

	client := api.NewRiotClient(key)
	internal.Title(fmt.Sprintf(i18n.T("top.title"), tierName, region))

	players, err := client.GetTopPlayers(region, "RANKED_SOLO_5x5", tier)
	if err != nil {
		internal.Error(fmt.Sprintf(i18n.T("error.fetch_failed"), err))
		os.Exit(1)
	}

	if len(players) == 0 {
		fmt.Println("  " + i18n.T("top.no_data"))
		return
	}

	// Sort by LP descending
	sort.Slice(players, func(i, j int) bool {
		return players[i].LeaguePoints > players[j].LeaguePoints
	})

	if limit > 0 && limit < len(players) {
		players = players[:limit]
	}

	headersStr := i18n.T("top.headers")
	headers := strings.Split(headersStr, ",")
	var rows [][]string

	tierColor := map[string]string{
		"CHALLENGER":   "\033[33m",
		"GRANDMASTER":   "\033[31m",
		"MASTER":        "\033[35m",
	}

	for i, p := range players {
		winRate := api.WinRate(p.Wins, p.Losses)
		wrStr := fmt.Sprintf("%.1f%%", winRate)

		hotStreak := ""
		if p.HotStreak {
			hotStreak = "🔥"
		}

		name := p.SummonerName
		if name == "" {
			name = "(隐藏)"
		}

		color := tierColor[p.Tier]
		if color == "" {
			color = "\033[37m"
		}

		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			fmt.Sprintf("%s%s\033[0m", color, name),
			fmt.Sprintf("%d LP", p.LeaguePoints),
			fmt.Sprintf("%d/%d", p.Wins, p.Losses),
			wrStr,
			hotStreak,
		})
	}

	internal.Table(headers, rows)
}
