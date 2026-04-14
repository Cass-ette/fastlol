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

var itemsCmd = &cobra.Command{
	Use:   "items <champion>",
	Short: "Show recommended items/build for a champion",
	Long: `Show recommended starting items, core build, and boots for a champion.

Examples:
  fastlol items gwen
  fastlol items gwen --role top`,
	Args: cobra.ExactArgs(1),
	Run:  runItems,
}

func init() {
	itemsCmd.Flags().StringP("role", "", "", "Filter by role (top, jg, mid, adc, sup)")
	rootCmd.AddCommand(itemsCmd)
}

func runItems(cmd *cobra.Command, args []string) {
	champion := args[0]
	role, _ := cmd.Flags().GetString("role")

	internal.Title(fmt.Sprintf(i18n.T("items.title"), champion))

	// Check for custom server
	serverURL := viper.GetString("api_url")

	if serverURL != "" {
		// Try to get items from custom server
		// For now, fall back to local scraper
		getItemsLocal(champion, role)
		return
	}

	getItemsLocal(champion, role)
}

func getItemsLocal(champion, role string) {
	scraper := api.NewUGGScraper()
	runes, err := scraper.GetRunes(champion, role)
	if err != nil {
		internal.Error(fmt.Sprintf(i18n.T("error.fetch_failed"), err))
		os.Exit(1)
	}

	displayItems(runes)
}

func displayItems(data *api.RuneData) {
	// Show data source info
	if data.Role != "" {
		fmt.Printf("  \033[90m[%s %s | 段位 Emerald+]\033[0m\n\n", data.Champion, data.Role)
	}

	// Show win rate and sample size
	if data.WinRate > 0 {
		wr := internal.WinRateColor(data.WinRate)
		fmt.Printf("  %s: %s\n", i18n.T("runes.win_rate"), wr)
	}
	if data.SampleSize > 0 {
		fmt.Printf("  %s: %d", i18n.T("runes.sample"), data.SampleSize)
		if data.SampleSize < 500 {
			fmt.Printf(" \033[33m(样本较少)\033[0m")
		}
		fmt.Println()
	}

	// Show starting items
	if len(data.StartingItems) > 0 {
		fmt.Printf("\n  \033[1m出门装:\033[0m\n    ")
		for i, item := range data.StartingItems {
			if i > 0 {
				fmt.Printf(" → ")
			}
			if item.Name != "" {
				fmt.Printf("%s", item.Name)
			} else {
				fmt.Printf("物品%d", item.ID)
			}
		}
		fmt.Println()
	}

	// Show core items
	if len(data.CoreItems) > 0 {
		fmt.Printf("\n  \033[1m核心装备:\033[0m\n    ")
		for i, item := range data.CoreItems {
			if i > 0 {
				fmt.Printf(" → ")
			}
			if item.Name != "" {
				fmt.Printf("%s", item.Name)
			} else {
				fmt.Printf("物品%d", item.ID)
			}
		}
		fmt.Println()
	}

	// Show skill order
	if len(data.SkillOrder) >= 6 {
		fmt.Printf("\n  \033[1m技能加点:\033[0m\n    ")
		// Show first 9 skill choices (full early game)
		for i := 0; i < 9 && i < len(data.SkillOrder); i++ {
			if i > 0 {
				fmt.Printf(" → ")
			}
			fmt.Printf("%s", data.SkillOrder[i])
		}
		fmt.Println()
	}

	// Show runes summary (abbreviated)
	if data.Keystone.Name != "" {
		fmt.Printf("\n  \033[1m符文:\033[0m %s | %s - %s\n",
			data.PrimaryTree, data.SecondaryTree, data.Keystone.Name)
	}

	// If no detailed data available
	if len(data.StartingItems) == 0 && len(data.CoreItems) == 0 {
		fmt.Println("\n  暂无出装数据")
	}
}
