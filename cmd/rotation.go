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

var rotationCmd = &cobra.Command{
	Use:   "rotation",
	Short: "View this week's free champion rotation",
	Long:  "View this week's free champion rotation for your region.",
	Args:  cobra.NoArgs,
	Run:   runRotation,
}

func init() {
	rotationCmd.Flags().StringP("region", "r", "", "Server region (e.g. kr, na1, euw1)")
	rootCmd.AddCommand(rotationCmd)
}

func runRotation(cmd *cobra.Command, args []string) {
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

	client := api.NewRiotClient(key)
	internal.Title(fmt.Sprintf(i18n.T("rotation.title"), region))

	rotation, err := client.GetChampionRotation(region)
	if err != nil {
		internal.Error(fmt.Sprintf(i18n.T("error.fetch_failed"), err))
		os.Exit(1)
	}

	fmt.Printf("  \033[1m"+i18n.T("rotation.header")+"\033[0m\n\n", len(rotation.FreeChampionIDs))

	var names []string
	for _, id := range rotation.FreeChampionIDs {
		names = append(names, getChampionName(id))
	}

	// Display in columns (4 per row)
	for i := 0; i < len(names); i += 4 {
		end := i + 4
		if end > len(names) {
			end = len(names)
		}
		row := names[i:end]
		formatted := make([]string, len(row))
		for j, n := range row {
			formatted[j] = fmt.Sprintf("%-16s", n)
		}
		fmt.Printf("  %s\n", strings.Join(formatted, "  "))
	}

	if len(rotation.FreeChampionIDsForNewPlayers) > 0 {
		fmt.Printf("\n  \033[90m"+i18n.T("rotation.newbie")+"\033[0m\n",
			len(rotation.FreeChampionIDsForNewPlayers),
			rotation.MaxNewPlayerLevel)
	}
}
