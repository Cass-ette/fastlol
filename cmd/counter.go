package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"fastlol/api"
	"fastlol/internal"
	"fastlol/internal/i18n"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var counterCmd = &cobra.Command{
	Use:   "counter <champion> [enemy]",
	Short: "Show counter picks and win rates for a champion",
	Long: `View counter picks and win rates for a champion.

Examples:
  fastlol counter gwen              # Show who counters Gwen
  fastlol counter gwen akali        # Show Gwen vs Akali matchup win rate`,
	Args:  cobra.RangeArgs(1, 2),
	Run:  runCounter,
}

func init() {
	counterCmd.Flags().StringP("role", "", "", "Filter by role (top, jg, mid, adc, sup)")
	counterCmd.Flags().BoolP("api", "", false, "Use RapidAPI instead of web scraping (requires API key)")
	counterCmd.Flags().BoolP("local", "", false, "Force use local scraper instead of custom server")
	rootCmd.AddCommand(counterCmd)
}

func runCounter(cmd *cobra.Command, args []string) {
	champion := args[0]
	enemy := ""
	if len(args) >= 2 {
		enemy = args[1]
	}
	role, _ := cmd.Flags().GetString("role")
	useAPI, _ := cmd.Flags().GetBool("api")
	useLocal, _ := cmd.Flags().GetBool("local")

	// Check for custom server
	serverURL := viper.GetString("api_url")

	// Two champion names = specific matchup query
	if enemy != "" {
		internal.Title(fmt.Sprintf("%s vs %s", champion, enemy))
		runMatchup(champion, enemy, role, useAPI, useLocal, serverURL)
		return
	}

	internal.Title(fmt.Sprintf(i18n.T("counter.title"), champion))

	// Priority: --local > api_url > --api > default (local scraper)
	if useLocal || (serverURL == "" && !useAPI) {
		// Use local scraper (default)
		data, err := scrapeCounters(champion, role)
		if err != nil {
			internal.Error(fmt.Sprintf("Scraping failed: %v", err))
			os.Exit(1)
		}
		displayScrapedData(data)
		return
	}

	if serverURL != "" {
		// Use custom server
		runCounterFromServer(champion, role, serverURL)
		return
	}

	// Use RapidAPI only if explicitly requested with --api flag
	key := viper.GetString("rapidapi_key")
	if key == "" {
		internal.Error("RapidAPI key not set. Use --api flag only if you have a key.")
		fmt.Fprintf(os.Stderr, "  Set key: fastlol config set rapidapi_key <your-key>\n")
		os.Exit(1)
	}

	client := api.NewClient(key)

	// Try counter_stat endpoint first
	data, err := client.GetCounterStat(champion, role)
	if err != nil {
		// Fallback to champ_stat endpoint
		data, err = client.GetChampionStat(champion)
		if err != nil {
			internal.Error(fmt.Sprintf(i18n.T("error.fetch_failed"), err))
			os.Exit(1)
		}
	}

	displayCounterData(champion, data)
}

func runCounterFromServer(champion, role, serverURL string) {
	client := api.NewServerClient(serverURL)
	data, err := client.GetCounters(champion, role)
	if err != nil {
		internal.Error(fmt.Sprintf("Server request failed: %v", err))
		os.Exit(1)
	}
	displayScrapedData(data)
}

func runMatchup(champion, enemy, role string, useAPI bool, useLocal bool, serverURL string) {
	// Priority: --local > api_url > --api > default (local scraper)
	if useLocal || (serverURL == "" && !useAPI) {
		// Use local scraper (default)
		scraper := api.NewMultiScraper()
		matchup, err := scraper.GetMatchup(champion, enemy, role)
		if err != nil {
			internal.Error(fmt.Sprintf("Matchup lookup failed: %v", err))
			os.Exit(1)
		}
		displayMatchup(matchup)
		return
	}

	if serverURL != "" {
		// Use custom server
		runMatchupFromServer(champion, enemy, role, serverURL)
		return
	}

	// Use RapidAPI only if explicitly requested
	key := viper.GetString("rapidapi_key")
	if key == "" {
		internal.Error("RapidAPI key not set. Use --api flag only if you have a key.")
		fmt.Fprintf(os.Stderr, "  Set key: fastlol config set rapidapi_key <your-key>\n")
		os.Exit(1)
	}

	// Fallback: try to extract from RapidAPI counter list
	client := api.NewClient(key)
	data, err := client.GetCounterStat(champion, role)
	if err != nil {
		internal.Error(fmt.Sprintf(i18n.T("error.fetch_failed"), err))
		os.Exit(1)
	}

	// Search for enemy in the counter list
	found := extractMatchupFromCounterList(data, enemy)
	if found == nil {
		fmt.Printf("  No matchup data found for %s vs %s\n", champion, enemy)
		return
	}
	displayMatchup(found)
}

func runMatchupFromServer(champion, enemy, role, serverURL string) {
	client := api.NewServerClient(serverURL)
	matchup, err := client.GetMatchup(champion, enemy, role)
	if err != nil {
		internal.Error(fmt.Sprintf("Server request failed: %v", err))
		os.Exit(1)
	}
	displayMatchup(matchup)
}

func extractMatchupFromCounterList(data json.RawMessage, enemy string) *api.MatchupResult {
	var counters api.ChampionCounter
	if err := json.Unmarshal(data, &counters); err == nil {
		for _, m := range counters.WeakAgainst {
			if strings.EqualFold(m.Name, enemy) {
				return &api.MatchupResult{
					Champion: m.Name,
					WinRate:  m.WinRate,
					SampleSize: 0,
				}
			}
		}
		for _, m := range counters.StrongAgainst {
			if strings.EqualFold(m.Name, enemy) {
				return &api.MatchupResult{
					Champion: m.Name,
					WinRate:  m.WinRate,
					SampleSize: 0,
				}
			}
		}
	}
	return nil
}

func displayMatchup(matchup *api.MatchupResult) {
	wr := matchup.WinRate
	if wr > 1 {
		wr = wr / 100
	}

	color := "\033[32m" // Green (good)
	if wr < 0.48 {
		color = "\033[31m" // Red (bad)
	} else if wr < 0.52 {
		color = "\033[33m" // Yellow (even)
	}

	fmt.Printf("\n  Win Rate: %s%.1f%%\033[0m\n", color, wr*100)
	if matchup.SampleSize > 0 {
		fmt.Printf("  Sample: %d games\n", matchup.SampleSize)
	}
	fmt.Printf("  Source: %s\n", matchup.Source)
}

func scrapeCounters(champion, role string) (*api.CounterData, error) {
	// Use UGGScraper directly to avoid fallback to other sources
	// This ensures specific errors (like "insufficient data") are passed through
	scraper := api.NewUGGScraper()
	return scraper.GetCounters(champion, role)
}

func displayScrapedData(data *api.CounterData) {
	// Show data source info
	if data.Version != "" && data.Tier != "" {
		matchupType := data.Role
		if matchupType == "" {
			matchupType = "Top"
		}
		fmt.Printf("  \033[90m[版本 %s | 段位 %s | %s vs %s | 最少200局]\033[0m\n\n", data.Version, data.Tier, data.Champion, matchupType)
	}

	if len(data.WeakAgainst) > 0 {
		fmt.Println(i18n.T("counter.weak_against"))
		h := strings.Split(i18n.T("counter.headers_weak"), ",")
		headers := []string{h[0], h[1]}
		var rows [][]string
		for _, m := range data.WeakAgainst {
			wr := internal.WinRateColor(m.WinRate)
			if m.WinRate >= 1 {
				wr = internal.WinRateColorPct(m.WinRate)
			}
			rows = append(rows, []string{m.Name, wr})
		}
		internal.Table(headers, rows)
	}

	if len(data.StrongAgainst) > 0 {
		fmt.Println()
		fmt.Println(i18n.T("counter.strong_against"))
		h := strings.Split(i18n.T("counter.headers_strong"), ",")
		headers := []string{h[0], h[1]}
		var rows [][]string
		for _, m := range data.StrongAgainst {
			wr := internal.WinRateColor(m.WinRate)
			if m.WinRate >= 1 {
				wr = internal.WinRateColorPct(m.WinRate)
			}
			rows = append(rows, []string{m.Name, wr})
		}
		internal.Table(headers, rows)
	}

	if len(data.WeakAgainst) == 0 && len(data.StrongAgainst) == 0 {
		fmt.Println("  " + i18n.T("tier.no_data"))
	}
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

	// Try as array (counter_stat may return array)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err == nil && len(arr) > 0 {
		displayCounterArray(arr)
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

func displayCounterArray(arr []json.RawMessage) {
	// Parse each entry as a generic object and display
	h := strings.Split(i18n.T("counter.headers.basic"), ",")
	headers := []string{h[0], h[1], h[2]}
	var rows [][]string

	for _, item := range arr {
		var m map[string]interface{}
		if err := json.Unmarshal(item, &m); err != nil {
			continue
		}
		name := ""
		for _, k := range []string{"name", "championName", "champion_name", "champion"} {
			if v, ok := m[k]; ok {
				name = fmt.Sprintf("%v", v)
				break
			}
		}
		if name == "" {
			continue
		}
		wr := ""
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
		games := ""
		for _, k := range []string{"games", "totalGames", "total_games", "count"} {
			if v, ok := m[k]; ok {
				games = fmt.Sprintf("%v", v)
				break
			}
		}
		rows = append(rows, []string{name, wr, games})
	}

	if len(rows) > 0 {
		internal.Table(headers, rows)
	}
}

func displayStructuredCounter(counter api.ChampionCounter) {
	if len(counter.WeakAgainst) > 0 {
		fmt.Println(i18n.T("counter.weak_against"))
		h := strings.Split(i18n.T("counter.headers_weak"), ",")
		headers := []string{h[0], h[1]}
		var rows [][]string
		for _, m := range counter.WeakAgainst {
			rows = append(rows, []string{m.Name, internal.WinRateColor(m.WinRate)})
		}
		internal.Table(headers, rows)
	}

	if len(counter.StrongAgainst) > 0 {
		fmt.Println()
		fmt.Println(i18n.T("counter.strong_against"))
		h := strings.Split(i18n.T("counter.headers_strong"), ",")
		headers := []string{h[0], h[1]}
		var rows [][]string
		for _, m := range counter.StrongAgainst {
			rows = append(rows, []string{m.Name, internal.WinRateColor(m.WinRate)})
		}
		internal.Table(headers, rows)
	}
}

func displayGenericCounter(champion string, raw map[string]json.RawMessage) {
	counterKeys := []string{"counter", "counters", "counterPicks", "counter_picks", "weakAgainst", "weak_against"}
	strongKeys := []string{"strongAgainst", "strong_against", "goodAgainst", "good_against", "easyMatchups", "easy_matchups"}
	statKeys := []string{"winRate", "win_rate", "pickRate", "pick_rate", "banRate", "ban_rate", "tier", "kda"}

	foundStats := false
	for _, key := range statKeys {
		if val, ok := raw[key]; ok {
			if !foundStats {
				fmt.Println("\033[1m  " + i18n.T("counter.stats") + ":\033[0m")
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

	for _, key := range counterKeys {
		if val, ok := raw[key]; ok {
			fmt.Println(i18n.T("counter.weak_against"))
			printMatchupList(val)
			fmt.Println()
		}
	}

	for _, key := range strongKeys {
		if val, ok := raw[key]; ok {
			fmt.Println(i18n.T("counter.strong_against"))
			printMatchupList(val)
			fmt.Println()
		}
	}

	if !foundStats {
		fmt.Printf("  返回字段: ")
		first := true
		for k := range raw {
			if !first {
				fmt.Print(", ")
			}
			fmt.Print(k)
			first = false
		}
		fmt.Println()
		fmt.Println("\n  原始数据 (前500字符):")
		out, _ := json.MarshalIndent(raw, "  ", "  ")
		s := string(out)
		if len(s) > 500 {
			s = s[:500] + "..."
		}
		fmt.Println(s)
	}
}

func printMatchupList(data json.RawMessage) {
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

	var names []string
	if err := json.Unmarshal(data, &names); err == nil {
		for _, n := range names {
			fmt.Printf("    %s\n", n)
		}
		return
	}

	fmt.Printf("    %s\n", string(data))
}
