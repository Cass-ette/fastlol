package cmd

import (
	"fmt"
	"os"
	"strings"

	"fastlol/api"

	"github.com/spf13/cobra"
)

const lcuDiscoveryHint = "Hint: start League Client, or pass --lockfile <path>; for manual sessions, pass --lcu-port with FASTLOL_LCU_TOKEN."

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Read local League Client status",
	Long:  "Read local League Client status from the read-only LCU API.",
}

var clientStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show League Client connection status",
	Args:  cobra.NoArgs,
	Run:   runClientStatus,
}

var clientCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current local summoner",
	Args:  cobra.NoArgs,
	Run:   runClientCurrent,
}

var clientGameflowCmd = &cobra.Command{
	Use:   "gameflow",
	Short: "Show current League Client gameflow phase",
	Args:  cobra.NoArgs,
	Run:   runClientGameflow,
}

var clientTeamCmd = &cobra.Command{
	Use:   "team",
	Short: "Show current lobby or champ-select teammates",
	Long:  "Show current lobby or champ-select teammates from the read-only local LCU API.",
	Args:  cobra.NoArgs,
	Run:   runClientTeam,
}

func init() {
	clientCmd.PersistentFlags().Int("lcu-port", 0, "LCU API port for manual session auth")
	clientCmd.PersistentFlags().String("lockfile", "", "Path to League Client lockfile")
	clientCmd.AddCommand(clientStatusCmd, clientCurrentCmd, clientGameflowCmd, clientTeamCmd)
	rootCmd.AddCommand(clientCmd)
}

func mustLocalLCUClient(cmd *cobra.Command) (*api.LCUClient, api.LCUConnectionInfo) {
	lcuPort, _ := cmd.Flags().GetInt("lcu-port")
	lockfile, _ := cmd.Flags().GetString("lockfile")

	info, err := api.DiscoverLCU(api.LCUDiscoveryOptions{Port: lcuPort, LockfilePath: lockfile})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not discover League Client: %v\n%s\n", err, lcuDiscoveryHint)
		os.Exit(1)
	}
	return api.NewLCUClient(info), info
}

func runClientStatus(cmd *cobra.Command, args []string) {
	client, info := mustLocalLCUClient(cmd)
	phase, err := getLCUGameflowPhase(client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read League Client gameflow phase: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Connected: yes")
	fmt.Printf("Source: %s\n", info.RedactedString())
	fmt.Printf("Phase: %s\n", phase)
}

func runClientCurrent(cmd *cobra.Command, args []string) {
	client, _ := mustLocalLCUClient(cmd)

	var summoner lcuCurrentSummoner
	if err := client.GetJSON("/lol-summoner/v1/current-summoner", &summoner); err != nil {
		fmt.Fprintf(os.Stderr, "Could not read current summoner: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Summoner: %s\n", summoner.Name())
	if summoner.SummonerLevel != nil {
		fmt.Printf("Level: %d\n", *summoner.SummonerLevel)
	}
	if shortPUUID := shortenPUUID(summoner.PUUID); shortPUUID != "" {
		fmt.Printf("PUUID: %s\n", shortPUUID)
	}
}

func runClientGameflow(cmd *cobra.Command, args []string) {
	client, info := mustLocalLCUClient(cmd)
	phase, err := getLCUGameflowPhase(client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read League Client gameflow phase: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Phase: %s\n", phase)
	fmt.Printf("Source: %s\n", info.RedactedString())
}

func runClientTeam(cmd *cobra.Command, args []string) {
	client, info := mustLocalLCUClient(cmd)
	snapshot, err := client.GetCurrentTeam()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read League Client team: %v\n", err)
		fmt.Fprintln(os.Stderr, "Hint: enter a lobby or champ select, then try again.")
		os.Exit(1)
	}

	fmt.Printf("Phase: %s\n", snapshot.Phase)
	fmt.Printf("LCU Source: %s\n", info.RedactedString())
	fmt.Printf("Team Source: %s\n", snapshot.Source)
	if len(snapshot.Members) == 0 {
		fmt.Println("No teammates found.")
		return
	}

	for i, member := range snapshot.Members {
		marker := ""
		if member.LocalPlayer {
			marker = " (you)"
		}
		fmt.Printf("%d. %s%s\n", i+1, member.DisplayName(), marker)
		if member.AssignedPosition != "" {
			fmt.Printf("   Position: %s\n", member.AssignedPosition)
		}
		if member.ChampionID > 0 {
			champion := member.ChampionName
			if champion == "" {
				champion = fmt.Sprintf("ID:%d", member.ChampionID)
			}
			fmt.Printf("   Champion: %s\n", champion)
		}
		if shortPUUID := shortenPUUID(member.PUUID); shortPUUID != "" {
			fmt.Printf("   PUUID: %s\n", shortPUUID)
		}
	}
}

func getLCUGameflowPhase(client *api.LCUClient) (string, error) {
	var phase string
	if err := client.GetJSON("/lol-gameflow/v1/gameflow-phase", &phase); err != nil {
		return "", err
	}
	if strings.TrimSpace(phase) == "" {
		return "Unknown", nil
	}
	return phase, nil
}

type lcuCurrentSummoner struct {
	DisplayName   string `json:"displayName"`
	GameName      string `json:"gameName"`
	TagLine       string `json:"tagLine"`
	SummonerLevel *int   `json:"summonerLevel"`
	PUUID         string `json:"puuid"`
}

func (s lcuCurrentSummoner) Name() string {
	if s.DisplayName != "" {
		return s.DisplayName
	}
	if s.GameName != "" && s.TagLine != "" {
		return s.GameName + "#" + s.TagLine
	}
	if s.GameName != "" {
		return s.GameName
	}
	return "Unknown"
}

func shortenPUUID(puuid string) string {
	puuid = strings.TrimSpace(puuid)
	if len(puuid) <= 12 {
		return puuid
	}
	return puuid[:12]
}
