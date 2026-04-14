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

var tierCmd = &cobra.Command{
	Use:   "tier",
	Short: "Show current patch tier list for all champions",
	Long: `Show champion tier rankings for the current patch (requires RapidAPI key).

Examples:
  fastlol tier
  fastlol tier --role mid
  fastlol tier --role top -n 10
`,
	Run:   runTier,
}

func init() {
	tierCmd.Flags().StringP("role", "r", "", "Filter by role (top, jg, mid, adc, sup)")
	tierCmd.Flags().IntP("limit", "n", 20, "Number of champions to show")
	rootCmd.AddCommand(tierCmd)
}

func runTier(cmd *cobra.Command, args []string) {
	key := viper.GetString("rapidapi_key")
	if key == "" {
		internal.Error(i18n.T("error.no_rapid_key"))
		fmt.Fprintln(os.Stderr, fmt.Sprintf(i18n.T("error.set_key_hint"), "rapidapi_key"))
		os.Exit(1)
	}

	role, _ := cmd.Flags().GetString("role")
	limit, _ := cmd.Flags().GetInt("limit")
	client := api.NewClient(key)

	internal.Title(i18n.T("tier.title"))

	rankings, err := client.GetRanking()
	if err != nil {
		internal.Error(fmt.Sprintf(i18n.T("error.fetch_failed"), err))
		os.Exit(1)
	}

	// Filter by role if specified
	if role != "" {
		roleMap := map[string]string{
			"top": "TOP", "jg": "JUNGLE", "jungle": "JUNGLE",
			"mid": "MID", "middle": "MID",
			"adc": "ADC", "bot": "ADC",
			"sup": "SUPPORT", "support": "SUPPORT",
		}
		target, ok := roleMap[role]
		if !ok {
			target = role
		}
		var filtered []api.ChampionRanking
		for _, r := range rankings {
			if r.Role == target || r.Role == role {
				filtered = append(filtered, r)
			}
		}
		rankings = filtered
		if len(rankings) == 0 {
			internal.Warn(fmt.Sprintf(i18n.T("tier.no_data"), role))
			return
		}
	}

	// Sort by win rate descending
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].WinRate > rankings[j].WinRate
	})

	if limit > 0 && limit < len(rankings) {
		rankings = rankings[:limit]
	}

	headersStr := i18n.T("tier.headers")
	headers := strings.Split(headersStr, ",")
	var rows [][]string
	for i, r := range rankings {
		wr := internal.WinRateColorPct(r.WinRate)
		if r.WinRate < 1 {
			wr = internal.WinRateColor(r.WinRate)
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			r.Name,
			r.Tier,
			r.Role,
			wr,
			fmt.Sprintf("%.1f%%", r.PickRate),
			fmt.Sprintf("%.1f%%", r.BanRate),
		})
	}
	internal.Table(headers, rows)
}
