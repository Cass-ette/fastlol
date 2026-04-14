package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"fastlol/api"
	"fastlol/internal"
	"fastlol/internal/i18n"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var liveCmd = &cobra.Command{
	Use:   "live [name] [tag]",
	Short: "Check if summoner is in an active game",
	Long:  "View real-time game data for an ongoing match.",
	Args:  cobra.RangeArgs(1, 2),
	Run:   runLive,
}

func init() {
	liveCmd.Flags().StringP("region", "r", "", "Server region (e.g. kr, na1, euw1)")
	rootCmd.AddCommand(liveCmd)
}

func runLive(cmd *cobra.Command, args []string) {
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
	internal.Title(fmt.Sprintf(i18n.T("live.title"), displayName, region))

	account, err := client.GetAccountByTag(region, gameName, tagLine)
	if err != nil {
		internal.Error(fmt.Sprintf(i18n.T("error.not_found"), err))
		os.Exit(1)
	}

	game, err := client.GetActiveGame(region, account.PUUID)
	if err != nil {
		fmt.Println("  \033[1;32m" + displayName + "\033[0m\n\n")
		fmt.Println("  " + i18n.T("live.not_in_game"))
		fmt.Println("  " + i18n.T("live.tip"))
		return
	}

	fmt.Printf("\n  \033[1;32m%s#%s\033[0m " + i18n.T("live.in_game") + "\n\n", account.GameName, account.TagLine, account.GameName, account.TagLine)

	// Game info
	modeNames := map[string]string{
		"CLASSIC":    i18n.GetLocalizedMode("CLASSIC"),
		"ARAM":       i18n.GetLocalizedMode("ARAM"),
		"URF":        i18n.GetLocalizedMode("URF"),
		"ONEFORALL":  i18n.GetLocalizedMode("ONEFORALL"),
		"NEXUSBLITZ": i18n.GetLocalizedMode("NEXUSBLITZ"),
		"CHERRY":     i18n.GetLocalizedMode("CHERRY"),
	}
	modeName := modeNames[game.GameMode]
	if modeName == "" {
		modeName = game.GameMode
	}

	elapsed := ""
	if game.GameLength > 0 {
		elapsed = fmt.Sprintf(" | %s %s", i18n.T("live.elapsed"), api.FormatDuration(int(game.GameLength)))
	} else if game.GameStartTime > 0 {
		dur := time.Since(time.UnixMilli(game.GameStartTime))
		elapsed = fmt.Sprintf(" | %s %d:%02d", i18n.T("live.elapsed"), int(dur.Minutes()), int(dur.Seconds())%60)
	}

	fmt.Printf("  %s: %s%s\n\n", i18n.T("live.mode"), modeName, elapsed)

	// Show bans if any
	if len(game.BannedChampions) > 0 {
		team1Bans := []string{}
		team2Bans := []string{}
		for _, b := range game.BannedChampions {
			if b.ChampionID <= 0 {
				continue
			}
			name := getChampionName(int(b.ChampionID))
			if b.TeamID == 100 {
				team1Bans = append(team1Bans, name)
			} else {
				team2Bans = append(team2Bans, name)
			}
		}
		if len(team1Bans) > 0 || len(team2Bans) > 0 {
			fmt.Printf("  %s: %s\n", i18n.T("live.bans_blue"), strings.Join(team1Bans, ", "))
			fmt.Printf("  %s: %s\n\n", i18n.T("live.bans_red"), strings.Join(team2Bans, ", "))
		}
	}

	// Separate teams
	var team1, team2 []api.ActiveGameParticipant
	for _, p := range game.Participants {
		if p.TeamID == 100 {
			team1 = append(team1, p)
		} else {
			team2 = append(team2, p)
		}
	}

	printTeam := func(label string, players []api.ActiveGameParticipant) {
		fmt.Printf("  \033[1m%s\033[0m\n", label)
		headersStr := i18n.T("live.headers")
		headers := []string{}
		for _, h := range strings.Split(headersStr, ",") {
			headers = append(headers, strings.TrimSpace(h))
		}
		var rows [][]string
		for _, p := range players {
			champName := getChampionName(int(p.ChampionID))
			playerName := p.RiotID
			if playerName == "" {
				playerName = p.SummonerName
			}
			if playerName == "" {
				playerName = i18n.T("live.hidden")
			}
			s1 := i18n.GetLocalizedSpell(p.Spell1ID)
			s2 := i18n.GetLocalizedSpell(p.Spell2ID)
			rows = append(rows, []string{champName, playerName, s1 + "/" + s2})
		}
		internal.Table(headers, rows)
		fmt.Println()
	}

	printTeam(i18n.T("live.team_blue"), team1)
	printTeam(i18n.T("live.team_red"), team2)
}
