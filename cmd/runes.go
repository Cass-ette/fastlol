package cmd

import (
	"fmt"
	"os"

	"fastlol/api"
	"fastlol/internal"
	"fastlol/internal/i18n"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var runesCmd = &cobra.Command{
	Use:   "runes <champion>",
	Short: "Show recommended runes for a champion",
	Long: `Show recommended runes, keystone, and win rate for a champion.

Examples:
  fastlol runes gwen
  fastlol runes gwen --role top`,
	Args: cobra.ExactArgs(1),
	Run:  runRunes,
}

func init() {
	runesCmd.Flags().StringP("role", "", "", "Filter by role (top, jg, mid, adc, sup)")
	rootCmd.AddCommand(runesCmd)
}

func runRunes(cmd *cobra.Command, args []string) {
	champion := args[0]
	role, _ := cmd.Flags().GetString("role")

	internal.Title(fmt.Sprintf(i18n.T("runes.title"), champion))

	// Check for custom server
	serverURL := viper.GetString("api_url")

	if serverURL != "" {
		// Try to get runes from custom server
		// For now, fall back to local scraper
		getRunesLocal(champion, role)
		return
	}

	getRunesLocal(champion, role)
}

func getRunesLocal(champion, role string) {
	scraper := api.NewUGGScraper()
	runes, err := scraper.GetRunes(champion, role)
	if err != nil {
		internal.Error(fmt.Sprintf(i18n.T("error.fetch_failed"), err))
		os.Exit(1)
	}

	displayRunes(runes)
}

func displayRunes(data *api.RuneData) {
	// Show data source info
	if data.Role != "" {
		fmt.Printf("  \033[90m[段位 Emerald+ | %s %s]\033[0m\n\n", data.Champion, data.Role)
	}

	// Show win rate and sample size
	if data.WinRate > 0 {
		wr := internal.WinRateColor(data.WinRate)
		fmt.Printf("  %s: %s\n", i18n.T("runes.win_rate"), wr)
	}
	if data.PickRate > 0 {
		fmt.Printf("  %s: %.1f%%\n", i18n.T("runes.pick_rate"), data.PickRate*100)
	}
	if data.SampleSize > 0 {
		fmt.Printf("  %s: %d\n", i18n.T("runes.sample"), data.SampleSize)
	}

	// Show primary rune tree
	if data.PrimaryTree != "" {
		fmt.Printf("\n  \033[1m%s:\033[0m %s\n", i18n.T("runes.primary"), data.PrimaryTree)
		if data.Keystone.Name != "" {
			fmt.Printf("    ⚡ %s\n", data.Keystone.Name)
		}
		for _, rune := range data.PrimaryRunes {
			fmt.Printf("    • %s\n", rune.Name)
		}
	}

	// Show secondary rune tree
	if data.SecondaryTree != "" {
		fmt.Printf("\n  \033[1m%s:\033[0m %s\n", i18n.T("runes.secondary"), data.SecondaryTree)
		for _, rune := range data.SecondaryRunes {
			fmt.Printf("    • %s\n", rune.Name)
		}
	}

	// Show shards
	if len(data.Shards) > 0 {
		fmt.Printf("\n  \033[1m%s:\033[0m\n", i18n.T("runes.shards"))
		for _, shard := range data.Shards {
			fmt.Printf("    • %s\n", shard)
		}
	}

	// If no detailed rune data available
	if data.PrimaryTree == "" && data.SecondaryTree == "" {
		fmt.Println("\n  " + i18n.T("runes.no_data"))
	}
}
