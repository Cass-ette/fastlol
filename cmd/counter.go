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

var counterCmd = &cobra.Command{
	Use:   "counter <champion>",
	Short: "Show counter picks for a champion",
	Long:  "Look up which champions are strong/weak against a given champion.",
	Args:  cobra.ExactArgs(1),
	Run:   runCounter,
}

func init() {
	rootCmd.AddCommand(counterCmd)
}

func runCounter(cmd *cobra.Command, args []string) {
	key := viper.GetString("rapidapi_key")
	if key == "" {
		internal.Error("No RapidAPI key configured.")
		fmt.Fprintln(os.Stderr, "Set it in ~/.fastlol/config.yaml or pass --rapidapi-key")
		os.Exit(1)
	}

	champion := args[0]
	client := api.NewClient(key)

	internal.Title(fmt.Sprintf("Counter data for: %s", champion))

	data, err := client.GetChampionStat(champion)
	if err != nil {
		internal.Error(fmt.Sprintf("Failed to fetch data: %v", err))
		os.Exit(1)
	}

	// Parse the response adaptively — API format may vary
	displayCounterData(champion, data)
}

func displayCounterData(champion string, data json.RawMessage) {
	// Try structured counter format first
	var counter api.ChampionCounter
	if err := json.Unmarshal(data, &counter); err == nil && len(counter.StrongAgainst) > 0 {
		displayStructuredCounter(counter)
		return
	}

	// Try as generic map to extract counter-related fields
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err == nil {
		displayGenericCounter(champion, raw)
		return
	}

	// Fallback: pretty-print raw JSON
	var pretty interface{}
	if err := json.Unmarshal(data, &pretty); err == nil {
		out, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Println(string(data))
	}
}

func displayStructuredCounter(counter api.ChampionCounter) {
	if len(counter.WeakAgainst) > 0 {
		fmt.Println("\033[1;31m  Countered by (weak against):\033[0m")
		headers := []string{"Champion", "Win Rate vs You"}
		var rows [][]string
		for _, m := range counter.WeakAgainst {
			rows = append(rows, []string{m.Name, internal.WinRateColor(m.WinRate)})
		}
		internal.Table(headers, rows)
	}

	if len(counter.StrongAgainst) > 0 {
		fmt.Println()
		fmt.Println("\033[1;32m  Strong against:\033[0m")
		headers := []string{"Champion", "Your Win Rate"}
		var rows [][]string
		for _, m := range counter.StrongAgainst {
			rows = append(rows, []string{m.Name, internal.WinRateColor(m.WinRate)})
		}
		internal.Table(headers, rows)
	}
}

func displayGenericCounter(champion string, raw map[string]json.RawMessage) {
	// Look for common keys in the response
	counterKeys := []string{"counter", "counters", "counterPicks", "counter_picks", "weakAgainst", "weak_against"}
	strongKeys := []string{"strongAgainst", "strong_against", "goodAgainst", "good_against", "easyMatchups", "easy_matchups"}
	statKeys := []string{"winRate", "win_rate", "pickRate", "pick_rate", "banRate", "ban_rate", "tier", "kda"}

	// Print any stats we find
	foundStats := false
	for _, key := range statKeys {
		if val, ok := raw[key]; ok {
			if !foundStats {
				fmt.Println("\033[1m  Stats:\033[0m")
				foundStats = true
			}
			var v interface{}
			json.Unmarshal(val, &v)
			fmt.Printf("    %s: %v\n", key, v)
		}
	}
	if foundStats {
		fmt.Println()
	}

	// Print counter matchups
	for _, key := range counterKeys {
		if val, ok := raw[key]; ok {
			fmt.Println("\033[1;31m  Countered by:\033[0m")
			printMatchupList(val)
			fmt.Println()
		}
	}

	for _, key := range strongKeys {
		if val, ok := raw[key]; ok {
			fmt.Println("\033[1;32m  Strong against:\033[0m")
			printMatchupList(val)
			fmt.Println()
		}
	}

	// If nothing matched, dump all keys
	if !foundStats {
		fmt.Printf("  Response keys: ")
		first := true
		for k := range raw {
			if !first {
				fmt.Print(", ")
			}
			fmt.Print(k)
			first = false
		}
		fmt.Println()
		fmt.Println("\n  Raw response (first 500 chars):")
		out, _ := json.MarshalIndent(raw, "  ", "  ")
		s := string(out)
		if len(s) > 500 {
			s = s[:500] + "..."
		}
		fmt.Println(s)
	}
}

func printMatchupList(data json.RawMessage) {
	// Try as array of objects
	var matchups []map[string]interface{}
	if err := json.Unmarshal(data, &matchups); err == nil {
		for _, m := range matchups {
			name := ""
			wr := ""
			for _, k := range []string{"name", "championName", "champion_name", "champion"} {
				if v, ok := m[k]; ok {
					name = fmt.Sprintf("%v", v)
					break
				}
			}
			for _, k := range []string{"winRate", "win_rate", "winrate"} {
				if v, ok := m[k]; ok {
					if f, ok := v.(float64); ok {
						if f < 1 {
							wr = internal.WinRateColor(f)
						} else {
							wr = internal.WinRateColorPct(f)
						}
					}
					break
				}
			}
			if name != "" {
				fmt.Printf("    %-20s %s\n", name, wr)
			}
		}
		return
	}

	// Try as array of strings
	var names []string
	if err := json.Unmarshal(data, &names); err == nil {
		for _, n := range names {
			fmt.Printf("    %s\n", n)
		}
		return
	}

	// Fallback
	fmt.Printf("    %s\n", string(data))
}
