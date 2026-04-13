package cmd

import (
	"fmt"
	"os"
	"sort"

	"fastlol/api"
	"fastlol/internal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tierCmd = &cobra.Command{
	Use:   "tier",
	Short: "Show current patch tier list",
	Run:   runTier,
}

func init() {
	tierCmd.Flags().StringP("role", "r", "", "Filter by role (top/jg/mid/adc/sup)")
	tierCmd.Flags().IntP("limit", "n", 20, "Number of champions to show")
	rootCmd.AddCommand(tierCmd)
}

func runTier(cmd *cobra.Command, args []string) {
	key := viper.GetString("rapidapi_key")
	if key == "" {
		internal.Error("No RapidAPI key configured.")
		fmt.Fprintln(os.Stderr, "Set it in ~/.fastlol/config.yaml or pass --rapidapi-key")
		os.Exit(1)
	}

	role, _ := cmd.Flags().GetString("role")
	limit, _ := cmd.Flags().GetInt("limit")
	client := api.NewClient(key)

	internal.Title("Tier List — Current Patch")

	rankings, err := client.GetRanking()
	if err != nil {
		internal.Error(fmt.Sprintf("Failed to fetch tier data: %v", err))
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
			internal.Warn(fmt.Sprintf("No champions found for role: %s", role))
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

	headers := []string{"#", "Champion", "Tier", "Role", "Win Rate", "Pick Rate", "Ban Rate"}
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
