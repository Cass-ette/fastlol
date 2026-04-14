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

var clashCmd = &cobra.Command{
	Use:   "clash [name] [tag]",
	Short: "View Clash tournament records",
	Long:  "View a summoner's Clash tournament participation.",
	Args: cobra.RangeArgs(1, 2),
	Run:  runClash,
}

func init() {
	clashCmd.Flags().StringP("region", "r", "", "Server region (e.g. kr, na1, euw1)")
	rootCmd.AddCommand(clashCmd)
}

func runClash(cmd *cobra.Command, args []string) {
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
	internal.Title(fmt.Sprintf(i18n.T("clash.title"), displayName, region))

	account, err := client.GetAccountByTag(region, gameName, tagLine)
	if err != nil {
		internal.Error(fmt.Sprintf(i18n.T("error.not_found"), err))
		os.Exit(1)
	}

	fmt.Printf("\n  \033[1;32m%s#%s\033[0m\n\n", account.GameName, account.TagLine)

	players, err := client.GetClashPlayers(region, account.PUUID)
	if err != nil {
		internal.Warn(fmt.Sprintf(i18n.T("clash.warn_fetch_failed"), err))
		hintLines := strings.Split(i18n.T("clash.hint_reasons"), "|")
		fmt.Println("\033[90m  " + i18n.T("clash.hint_title"))
		for _, line := range hintLines {
			fmt.Println("  - " + line)
		}
		fmt.Print("\033[0m")
		return
	}

	if len(players) == 0 {
		fmt.Println("  " + i18n.T("clash.no_data"))
		return
	}

	headersStr := i18n.T("clash.headers")
	headers := strings.Split(headersStr, ",")
	var rows [][]string

	posNames := map[string]string{
		"TOP":    i18n.GetLocalizedRole("TOP"),
		"JGL":    i18n.GetLocalizedRole("JGL"),
		"MID":    i18n.GetLocalizedRole("MID"),
		"ADC":    i18n.GetLocalizedRole("ADC"),
		"SUP":    i18n.GetLocalizedRole("SUP"),
		"NONE":   i18n.GetLocalizedRole("NONE"),
		"FILTER": i18n.GetLocalizedRole("FILTER"),
	}

	for _, p := range players {
		posName := posNames[p.Position]
		if posName == "" {
			posName = p.Position
		}

		name := p.SummonerName
		if name == "" {
			name = i18n.T("pos.hidden")
		}

		teamID := p.TeamID
		if teamID == "" {
			teamID = i18n.T("pos.no_team")
		}

		rows = append(rows, []string{posName, name, teamID})
	}

	internal.Table(headers, rows)

	fmt.Println()
	fmt.Println("  \033[90m"+i18n.T("clash.tip")+"\033[0m")
}
