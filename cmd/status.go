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

var statusCmd = &cobra.Command{
	Use:   "status [region]",
	Short: "Check server status (maintenance, incidents)",
	Long: `Check League server status and maintenance announcements.

Examples:
  fastlol status
  fastlol status kr
  fastlol status na1
`,
	Args: cobra.RangeArgs(0, 1),
	Run:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) {
	key := viper.GetString("riot_api_key")
	if key == "" {
		internal.Error(i18n.T("error.no_riot_key"))
		fmt.Fprintln(os.Stderr, fmt.Sprintf(i18n.T("error.set_key_hint"), "riot_api_key"))
		os.Exit(1)
	}

	region := "kr"
	if len(args) >= 1 {
		region = strings.ToLower(args[0])
	} else {
		region = viper.GetString("default_region")
		if region == "" {
			region = "kr"
		}
	}

	client := api.NewRiotClient(key)
	internal.Title(fmt.Sprintf(i18n.T("status.title"), region))

	status, err := client.GetServerStatus(region)
	if err != nil {
		internal.Error(fmt.Sprintf(i18n.T("error.fetch_failed"), err))
		os.Exit(1)
	}

	fmt.Printf("\n  "+i18n.T("status.server")+"\n\n", status.Name)

	regionNames := map[string]string{
		"kr":   i18n.GetLocalizedRegion("kr"),
		"euw1": i18n.GetLocalizedRegion("euw1"),
		"eun1": i18n.GetLocalizedRegion("eun1"),
		"na1":  i18n.GetLocalizedRegion("na1"),
		"jp1":  i18n.GetLocalizedRegion("jp1"),
		"br1":  i18n.GetLocalizedRegion("br1"),
		"la1":  i18n.GetLocalizedRegion("la1"),
		"la2":  i18n.GetLocalizedRegion("la2"),
		"oc1":  i18n.GetLocalizedRegion("oc1"),
		"tr1":  i18n.GetLocalizedRegion("tr1"),
		"ru":   i18n.GetLocalizedRegion("ru"),
		"sg2":  i18n.GetLocalizedRegion("sg2"),
		"ph2":  i18n.GetLocalizedRegion("ph2"),
		"th2":  i18n.GetLocalizedRegion("th2"),
		"tw2":  i18n.GetLocalizedRegion("tw2"),
		"vn2":  i18n.GetLocalizedRegion("vn2"),
		"cn1":  i18n.GetLocalizedRegion("cn1"),
	}

	if status.Incidents != nil && len(status.Incidents) > 0 {
		fmt.Println("  \033[33m" + i18n.T("status.incidents") + "\033[0m\n")
		for _, inc := range status.Incidents {
			statusIcon := "🟡"
			if inc.Status == "RESOLVED" {
				statusIcon = "✅"
			} else if inc.Status == "CRITICAL" {
				statusIcon = "🔴"
			}
			fmt.Printf("  %s %s\n", statusIcon, inc.Title)
		}
		fmt.Println()
	}

	if status.Maintenances != nil && len(status.Maintenances) > 0 {
		fmt.Println("  \033[31m" + i18n.T("status.maintenance") + "\033[0m\n")
		for _, m := range status.Maintenances {
			statusStr := m.Status
			if statusStr == "SCHEDULED" {
				statusStr = i18n.T("status.status.scheduled")
			} else if statusStr == "IN_PROGRESS" {
				statusStr = i18n.T("status.status.progress")
			}
			fmt.Printf("  [%s] %s\n", statusStr, m.MaintenanceType)
			for _, t := range m.Titles {
				if t.Locale == "en_US" && t.Content != "" {
					fmt.Printf("    %s\n", t.Content)
				}
			}
		}
		fmt.Println()
	}

	if (status.Incidents == nil || len(status.Incidents) == 0) &&
		(status.Maintenances == nil || len(status.Maintenances) == 0) {
		fmt.Println("  \033[32m" + i18n.T("status.normal") + "\033[0m\n")
	}

	fmt.Printf("  " + i18n.T("status.region_code") + "\n", region)
	if name, ok := regionNames[region]; ok {
		fmt.Printf(" (%s)", name)
	}
	fmt.Println()
}
