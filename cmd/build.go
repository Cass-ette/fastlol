package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"fastlol/api"
	"fastlol/internal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var buildCmd = &cobra.Command{
	Use:   "build <champion>",
	Short: "Show recommended build, runes, and win rate for a champion",
	Args:  cobra.ExactArgs(1),
	Run:   runBuild,
}

func init() {
	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) {
	key := viper.GetString("rapidapi_key")
	if key == "" {
		internal.Error("No RapidAPI key configured.")
		fmt.Fprintln(os.Stderr, "Set it in ~/.fastlol/config.yaml or pass --rapidapi-key")
		os.Exit(1)
	}

	champion := args[0]
	client := api.NewClient(key)

	internal.Title(fmt.Sprintf("Build info for: %s", champion))

	data, err := client.GetChampionStat(champion)
	if err != nil {
		internal.Error(fmt.Sprintf("Failed to fetch data: %v", err))
		os.Exit(1)
	}

	displayBuildData(data)
}

func displayBuildData(data json.RawMessage) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		// Pretty-print as fallback
		var pretty interface{}
		json.Unmarshal(data, &pretty)
		out, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(out))
		return
	}

	// Items / build
	buildKeys := []string{"items", "build", "coreItems", "core_items", "recommendedItems", "recommended_items", "startingItems", "starting_items"}
	for _, key := range buildKeys {
		if val, ok := raw[key]; ok {
			fmt.Printf("\033[1m  %s:\033[0m\n", key)
			printItemList(val)
			fmt.Println()
		}
	}

	// Runes
	runeKeys := []string{"runes", "primaryRune", "primary_rune", "secondaryRune", "secondary_rune", "perks"}
	for _, key := range runeKeys {
		if val, ok := raw[key]; ok {
			fmt.Printf("\033[1m  %s:\033[0m\n", key)
			printGenericList(val)
			fmt.Println()
		}
	}

	// Spells
	spellKeys := []string{"spells", "summonerSpells", "summoner_spells"}
	for _, key := range spellKeys {
		if val, ok := raw[key]; ok {
			fmt.Printf("\033[1m  %s:\033[0m\n", key)
			printGenericList(val)
			fmt.Println()
		}
	}

	// Win rate / stats
	statKeys := []string{"winRate", "win_rate", "pickRate", "pick_rate", "banRate", "ban_rate", "tier", "kda", "role"}
	for _, key := range statKeys {
		if val, ok := raw[key]; ok {
			var v interface{}
			json.Unmarshal(val, &v)
			fmt.Printf("  %s: %v\n", key, v)
		}
	}
}

func printItemList(data json.RawMessage) {
	var items []interface{}
	if err := json.Unmarshal(data, &items); err == nil {
		for _, item := range items {
			switch v := item.(type) {
			case string:
				fmt.Printf("    - %s\n", v)
			case map[string]interface{}:
				name := ""
				for _, k := range []string{"name", "itemName", "item_name"} {
					if n, ok := v[k]; ok {
						name = fmt.Sprintf("%v", n)
						break
					}
				}
				if name != "" {
					fmt.Printf("    - %s\n", name)
				} else {
					out, _ := json.Marshal(v)
					fmt.Printf("    - %s\n", string(out))
				}
			default:
				fmt.Printf("    - %v\n", item)
			}
		}
		return
	}
	fmt.Printf("    %s\n", string(data))
}

func printGenericList(data json.RawMessage) {
	var items []interface{}
	if err := json.Unmarshal(data, &items); err == nil {
		for _, item := range items {
			switch v := item.(type) {
			case string:
				fmt.Printf("    - %s\n", v)
			case map[string]interface{}:
				out, _ := json.Marshal(v)
				fmt.Printf("    - %s\n", string(out))
			default:
				fmt.Printf("    - %v\n", item)
			}
		}
		return
	}
	fmt.Printf("    %s\n", string(data))
}
