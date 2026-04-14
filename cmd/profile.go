package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"fastlol/api"
	"fastlol/internal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Champion cache - loaded once from Riot Data Dragon
var (
	championCache    map[int]string
	championCacheOnce sync.Once
	championCacheErr error
)

func getChampionName(id int) string {
	championCacheOnce.Do(func() {
		championCache, championCacheErr = fetchChampionNames()
	})
	if championCacheErr != nil {
		return fmt.Sprintf("ID:%d", id)
	}
	name, ok := championCache[id]
	if !ok {
		return fmt.Sprintf("ID:%d", id)
	}
	return name
}

func fetchChampionNames() (map[int]string, error) {
	// Use latest patch version
	url := "https://ddragon.leagueoflegends.com/cdn/16.7.1/data/en_US/champion.json"
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Data map[string]struct {
			Key string `json:"key"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	result := make(map[int]string)
	for _, c := range data.Data {
		var id int
		fmt.Sscanf(c.Key, "%d", &id)
		result[id] = c.Name
	}
	return result, nil
}

var profileCmd = &cobra.Command{
	Use:   "profile [选手名] [TAG]",
	Short: "查询召唤师战绩（英雄成就 + 近期比赛）",
	Long: `查询召唤师战绩

用法示例:
  fastlol profile "Caps" EUW --region euw1 --matches 3
  fastlol profile "Bin" KR1 --region kr
  fastlol profile "Hide on bush" KR --region kr
  fastlol profile "ON" KR1 --region kr --mastery 3

Riot ID 格式: 名字#TAG
常见区域: kr(韩服) euw1(欧服) na1(美服)
`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runProfile,
}

func init() {
	profileCmd.Flags().StringP("region", "r", "", "服务器区域 (kr, euw1, na1, cn1)")
	profileCmd.Flags().IntP("matches", "n", 5, "显示近期比赛数量")
	profileCmd.Flags().IntP("mastery", "m", 5, "显示英雄成就数量")
	profileCmd.Flags().Bool("mock", false, "使用模拟数据（无需 API key）")
	rootCmd.AddCommand(profileCmd)
}

func runProfile(cmd *cobra.Command, args []string) {
	useMock, _ := cmd.Flags().GetBool("mock")

	key := viper.GetString("riot_api_key")
	if !useMock && key == "" {
		internal.Error("未配置 Riot API key")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "配置方法:")
		fmt.Fprintln(os.Stderr, "  1. 访问 https://developer.riotgames.com/")
		fmt.Fprintln(os.Stderr, "  2. 获取 Development API Key")
		fmt.Fprintln(os.Stderr, "  3. fastlol config set riot_api_key <你的key>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "或使用模拟数据测试:")
		fmt.Fprintf(os.Stderr, "  fastlol profile %s --mock\n", args[0])
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

	matchCount, _ := cmd.Flags().GetInt("matches")
	masteryCount, _ := cmd.Flags().GetInt("mastery")

	gameName := args[0]
	tagLine := ""
	if len(args) >= 2 {
		tagLine = args[1]
	}

	// Handle Riot ID format with #
	if tagLine == "" && strings.Contains(gameName, "#") {
		parts := strings.SplitN(gameName, "#", 2)
		gameName = parts[0]
		tagLine = parts[1]
	}

	displayName := gameName
	if tagLine != "" {
		displayName = fmt.Sprintf("%s#%s", gameName, tagLine)
	}

	if useMock {
		internal.Title(fmt.Sprintf("[模拟数据] %s (%s)", displayName, region))
		runMockProfile(displayName, region, matchCount, masteryCount)
		return
	}

	client := api.NewRiotClient(key)
	internal.Title(fmt.Sprintf("🔍 正在查询: %s (%s)", displayName, region))

	account, err := client.GetAccountByTag(region, gameName, tagLine)
	if err != nil {
		internal.Error(fmt.Sprintf("未找到该召唤师: %v", err))
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "💡 常见问题:")
		fmt.Fprintln(os.Stderr, "   - 请确认格式: fastlol profile \"名字\" TAG")
		fmt.Fprintln(os.Stderr, "   - 示例: fastlol profile \"Caps\" EUW --region euw1")
		fmt.Fprintln(os.Stderr, "   - 如果账号设置了隐私保护，部分数据可能无法获取")
		os.Exit(1)
	}

	displayAccountInfo(account)

	// Champion mastery
	if masteryCount > 0 {
		masteries, err := client.GetChampionMastery(region, account.PUUID, masteryCount)
		if err == nil && len(masteries) > 0 {
			displayChampionMastery(masteries)
		}
	}

	// Recent matches
	if matchCount > 0 {
		matchIDs, err := client.GetRecentMatches(region, account.PUUID, matchCount)
		if err == nil && len(matchIDs) > 0 {
			matches := fetchMatchDetails(client, region, account.PUUID, matchIDs)
			if len(matches) > 0 {
				displayRecentMatches(matches)
			}
		} else if err != nil {
			internal.Warn(fmt.Sprintf("无法获取近期比赛: %v", err))
		}
	}

	fmt.Println()
	fmt.Println("\033[90m  💡 Development API Key 限制: 召唤师等级和排位数据可能无法获取\033[0m")
}

func displayAccountInfo(a *api.Account) {
	fmt.Printf("\n  \033[1;32m%s#%s\033[0m\n", a.GameName, a.TagLine)
	fmt.Printf("  PUUID: %s...%s\n\n", a.PUUID[:8], a.PUUID[len(a.PUUID)-8:])
}

func displayChampionMastery(masteries []api.ChampionMastery) {
	fmt.Printf("  \033[1m🎮 英雄成就 (Champion Mastery):\033[0m\n\n")

	headers := []string{"排名", "英雄", "等级", "熟练度", "最近使用"}
	var rows [][]string

	for i, m := range masteries {
		lastPlayed := time.UnixMilli(m.LastPlayTime).Format("01/02")
		name := getChampionName(int(m.ChampionID))
		levelStars := strings.Repeat("⭐", m.ChampionLevel)
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			name,
			fmt.Sprintf("Lv%d %s", m.ChampionLevel, levelStars),
			formatPoints(m.ChampionPoints),
			lastPlayed,
		})
	}

	internal.Table(headers, rows)
	fmt.Println()
}

func formatPoints(p int) string {
	if p >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(p)/1000000)
	}
	if p >= 1000 {
		return fmt.Sprintf("%.1fK", float64(p)/1000)
	}
	return fmt.Sprintf("%d", p)
}

func fetchMatchDetails(client *api.RiotClient, region, puuid string, matchIDs []string) []api.MatchMetadata {
	var matches []api.MatchMetadata
	for _, id := range matchIDs {
		match, err := client.GetMatchInfo(region, id)
		if err != nil {
			continue
		}
		meta := match.GetPlayerMatchMetadata(puuid)
		if meta != nil {
			matches = append(matches, *meta)
		}
	}
	return matches
}

func displayRecentMatches(matches []api.MatchMetadata) {
	fmt.Printf("  \033[1m📊 近期比赛 (Recent Matches):\033[0m\n\n")

	headers := []string{"时间", "模式", "英雄", "KDA", "结果", "时长"}
	var rows [][]string

	for _, meta := range matches {
		t := time.UnixMilli(meta.GameCreation)
		timeStr := t.Format("01/02 15:04")

		name := getChampionNameByChampName(meta.ChampionName)

		kda := fmt.Sprintf("%d/%d/%d", meta.Kills, meta.Deaths, meta.Assists)
		if meta.Deaths == 0 {
			if meta.Kills+meta.Assists > 0 {
				kda += " (完美)"
			}
		} else {
			ratio := float64(meta.Kills+meta.Assists) / float64(meta.Deaths)
			kda += fmt.Sprintf(" (%.2f)", ratio)
		}

		result := "\033[31m败\033[0m"
		if meta.Win {
			result = "\033[32m胜\033[0m"
		}

		duration := api.FormatDuration(meta.GameDuration)
		gameMode := meta.GameMode
		if gameMode == "CLASSIC" {
			gameMode = "召唤师峡谷"
		} else if gameMode == "ARAM" {
			gameMode = "极地大乱斗"
		}

		rows = append(rows, []string{
			timeStr,
			gameMode,
			name,
			kda,
			result,
			duration,
		})
	}

	internal.Table(headers, rows)
}

// getChampionNameByChampName tries to match by champion name (capitalization-insensitive)
func getChampionNameByChampName(name string) string {
	championCacheOnce.Do(func() {
		if championCacheErr == nil && championCache == nil {
			championCache, championCacheErr = fetchChampionNames()
		}
	})
	if championCacheErr != nil || championCache == nil {
		return name
	}
	// Try exact match first
	for _, v := range championCache {
		if strings.EqualFold(v, name) {
			return v
		}
	}
	return name
}

func runMockProfile(displayName, region string, matchCount, masteryCount int) {
	parts := strings.Split(displayName, "#")
	gameName := parts[0]
	tagLine := ""
	if len(parts) > 1 {
		tagLine = parts[1]
	}

	fmt.Printf("\n  \033[1;32m%s#%s\033[0m\n", gameName, tagLine)
	fmt.Printf("  PUUID: MOCK-xxxx-xxxx...\n\n")

	// Mock champion mastery
	fmt.Printf("  \033[1m🎮 英雄成就 (Champion Mastery):\033[0m\n\n")
	headers := []string{"排名", "英雄", "等级", "熟练度", "最近使用"}
	var rows [][]string
	champions := []string{"Akali", "Yasuo", "Ahri", "Zed", "Lee Sin"}
	levels := []int{7, 6, 5, 5, 4}
	for i := 0; i < masteryCount && i < len(champions); i++ {
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			champions[i],
			fmt.Sprintf("Lv%d %s", levels[i], strings.Repeat("⭐", levels[i])),
			fmt.Sprintf("%.1fK", float64(300-50*i)/1),
			"04/14",
		})
	}
	internal.Table(headers, rows)
	fmt.Println()

	if matchCount > 0 {
		displayMockMatches(matchCount)
	}

	fmt.Println()
	fmt.Println("\033[90m  💡 使用真实 API key 可查看实际数据\033[0m")
}

func displayMockMatches(count int) {
	fmt.Printf("  \033[1m📊 近期比赛 (Recent Matches):\033[0m\n\n")

	headers := []string{"时间", "模式", "英雄", "KDA", "结果", "时长"}
	var rows [][]string

	champions := []string{"Ahri", "Yasuo", "Lee Sin", "Jinx", "Thresh"}
	modes := []string{"召唤师峡谷", "极地大乱斗", "召唤师峡谷"}
	now := time.Now()

	for i := 0; i < count && i < len(champions); i++ {
		t := now.Add(-time.Duration(i*2) * time.Hour)
		win := i%3 != 0
		kills := 5 + i*2
		deaths := 3 + i%4
		assists := 8 + i

		kda := fmt.Sprintf("%d/%d/%d", kills, deaths, assists)
		if deaths == 0 {
			kda += " (完美)"
		} else {
			ratio := float64(kills+assists) / float64(deaths)
			kda += fmt.Sprintf(" (%.2f)", ratio)
		}

		result := "\033[31m败\033[0m"
		if win {
			result = "\033[32m胜\033[0m"
		}

		rows = append(rows, []string{
			t.Format("01/02 15:04"),
			modes[i%len(modes)],
			champions[i],
			kda,
			result,
			fmt.Sprintf("%d:%02d", 30+i*2, 0),
		})
	}

	internal.Table(headers, rows)
}
