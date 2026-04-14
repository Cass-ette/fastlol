package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// CacheEntry stores cached scrape results with timestamp
type CacheEntry struct {
	Data      *CounterData
	Matchup   *MatchupResult
	Timestamp time.Time
}

// ScraperCache provides simple in-memory caching for scrape results
type ScraperCache struct {
	mu    sync.RWMutex
	store map[string]CacheEntry
	ttl   time.Duration
}

// NewScraperCache creates a new cache with specified TTL
func NewScraperCache(ttl time.Duration) *ScraperCache {
	return &ScraperCache{
		store: make(map[string]CacheEntry),
		ttl:   ttl,
	}
}

// Get retrieves cached counter data if not expired
func (c *ScraperCache) Get(key string) (*CounterData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.store[key]
	if !exists || time.Since(entry.Timestamp) > c.ttl {
		return nil, false
	}
	return entry.Data, true
}

// GetMatchup retrieves cached matchup data if not expired
func (c *ScraperCache) GetMatchup(key string) (*MatchupResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.store[key]
	if !exists || time.Since(entry.Timestamp) > c.ttl {
		return nil, false
	}
	return entry.Matchup, true
}

// Set stores counter data in cache
func (c *ScraperCache) Set(key string, data *CounterData) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store[key] = CacheEntry{
		Data:      data,
		Timestamp: time.Now(),
	}
}

// SetMatchup stores matchup data in cache
func (c *ScraperCache) SetMatchup(key string, matchup *MatchupResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store[key] = CacheEntry{
		Matchup:   matchup,
		Timestamp: time.Now(),
	}
}

// makeCacheKey creates a cache key for counter queries
func makeCacheKey(source, champion, role string) string {
	return fmt.Sprintf("%s:counters:%s:%s", source, champion, role)
}

// makeMatchupCacheKey creates a cache key for matchup queries
func makeMatchupCacheKey(source, champion, enemy, role string) string {
	return fmt.Sprintf("%s:matchup:%s:%s:%s", source, champion, enemy, role)
}

// CounterData represents scraped counter information
type CounterData struct {
	Champion      string    `json:"champion"`
	Role          string    `json:"role"`
	WeakAgainst   []Matchup `json:"weak_against"`
	StrongAgainst []Matchup `json:"strong_against"`
	Source        string    `json:"source"`
	Version       string    `json:"version"` // e.g., "16.7"
	Tier          string    `json:"tier"`    // e.g., "Emerald+"
}

// Matchup represents a single matchup with win rate
type Matchup struct {
	Name    string  `json:"name"`
	WinRate float64 `json:"win_rate"`
	Games   int     `json:"games,omitempty"`
}

// MatchupResult represents a specific 1v1 matchup result
type MatchupResult struct {
	Champion   string  `json:"champion"`
	Enemy      string  `json:"enemy"`
	WinRate    float64 `json:"win_rate"`    // Champion's win rate vs Enemy
	SampleSize int     `json:"sample_size"` // Number of games analyzed
	Source     string  `json:"source"`
}

// Scraper defines the interface for scraping counter data
type Scraper interface {
	GetCounters(champion, role string) (*CounterData, error)
	Name() string
}

// MatchupScraper supports specific 1v1 matchup queries
type MatchupScraper interface {
	Scraper
	GetMatchup(champion, enemy, role string) (*MatchupResult, error)
}

// LeagueOfGraphsScraper scrapes from leagueofgraphs.com
type LeagueOfGraphsScraper struct {
	httpClient *http.Client
}

func NewLeagueOfGraphsScraper() *LeagueOfGraphsScraper {
	return &LeagueOfGraphsScraper{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *LeagueOfGraphsScraper) Name() string {
	return "leagueofgraphs"
}

func (s *LeagueOfGraphsScraper) GetCounters(champion, role string) (*CounterData, error) {
	role = normalizeRole(role)
	url := fmt.Sprintf("https://www.leagueofgraphs.com/champions/counters/%s/%s",
		strings.ToLower(champion), role)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	data := &CounterData{
		Champion: champion,
		Role:     role,
		Source:   s.Name(),
	}

	// LeagueOfGraphs structure: look for tables with matchup data
	// The page has sections with "worst against" and "best against"

	// Try to find matchup sections by looking for header text
	doc.Find("h2, h3, .title, .section-title").Each(func(i int, s *goquery.Selection) {
		text := strings.ToLower(s.Text())

		// Look for the table/data following this header
		nextTable := s.NextAllFiltered("table").First()
		if nextTable.Length() == 0 {
			// Try parent container
			parent := s.Parent()
			nextTable = parent.Find("table").First()
		}

		if strings.Contains(text, "worst") || strings.Contains(text, "counter") || strings.Contains(text, "difficult") {
			data.WeakAgainst = extractMatchups(nextTable)
		} else if strings.Contains(text, "best") || strings.Contains(text, "easy") || strings.Contains(text, "strong") {
			data.StrongAgainst = extractMatchups(nextTable)
		}
	})

	// Alternative: look for specific container classes
	if len(data.WeakAgainst) == 0 && len(data.StrongAgainst) == 0 {
		// Try to find data by common CSS patterns
		doc.Find("[class*='counter'], [class*='matchup'], [class*='worst'], [class*='best']").Each(func(i int, s *goquery.Selection) {
			text := strings.ToLower(s.Text())
			if strings.Contains(text, "worst") || strings.Contains(text, "counter") {
				// Find rows in this container
				data.WeakAgainst = append(data.WeakAgainst, extractMatchupsFromContainer(s)...)
			} else if strings.Contains(text, "best") || strings.Contains(text, "easy") {
				data.StrongAgainst = append(data.StrongAgainst, extractMatchupsFromContainer(s)...)
			}
		})
	}

	if len(data.WeakAgainst) == 0 && len(data.StrongAgainst) == 0 {
		return nil, fmt.Errorf("no counter data found in page (layout may have changed)")
	}

	return data, nil
}

func extractMatchups(table *goquery.Selection) []Matchup {
	var matchups []Matchup

	table.Find("tr").Each(func(i int, row *goquery.Selection) {
		if i == 0 {
			return // Skip header
		}

		cells := row.Find("td")
		if cells.Length() < 2 {
			return
		}

		name := strings.TrimSpace(cells.Eq(0).Text())
		wrText := strings.TrimSpace(cells.Eq(1).Text())

		winRate := parseWinRate(wrText)
		games := 0
		if cells.Length() > 2 {
			games = parseGames(cells.Eq(2).Text())
		}

		if name != "" {
			matchups = append(matchups, Matchup{
				Name:    name,
				WinRate: winRate,
				Games:   games,
			})
		}
	})

	return matchups
}

func extractMatchupsFromContainer(container *goquery.Selection) []Matchup {
	var matchups []Matchup

	// Look for row-like structures
	container.Find(".row, .item, .champion-row, [class*='champion']").Each(func(i int, item *goquery.Selection) {
		name := item.Find("[class*='name'], .champion").First().Text()
		if name == "" {
			// Try to get text directly
			name = strings.TrimSpace(item.Text())
			// Split by whitespace and take first part as name
			parts := strings.Fields(name)
			if len(parts) > 0 {
				name = parts[0]
			}
		}

		wrText := item.Find("[class*='win'], [class*='rate'], [class*='percentage']").First().Text()
		winRate := parseWinRate(wrText)

		if name != "" && winRate > 0 {
			matchups = append(matchups, Matchup{
				Name:    name,
				WinRate: winRate,
			})
		}
	})

	return matchups
}

func parseWinRate(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "%", "")
	s = strings.ReplaceAll(s, ",", ".")
	f, _ := strconv.ParseFloat(s, 64)
	// Convert percentage to decimal if needed
	if f > 1 {
		f = f / 100
	}
	return f
}

func parseGames(s string) int {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")
	n, _ := strconv.Atoi(s)
	return n
}

func normalizeRole(role string) string {
	role = strings.ToLower(role)
	switch role {
	case "top":
		return "top"
	case "jungle", "jg", "jun":
		return "jungle"
	case "mid", "middle":
		return "mid"
	case "adc", "bot", "bottom":
		return "adc"
	case "support", "sup":
		return "support"
	default:
		return role
	}
}

// UGGScraper scrapes from u.gg using their API
type UGGScraper struct {
	httpClient *http.Client
}

func NewUGGScraper() *UGGScraper {
	return &UGGScraper{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *UGGScraper) Name() string {
	return "ugg"
}

// championIDMap maps champion names to their IDs
var championIDMap = map[string]string{
	"aatrox": "266", "ahri": "103", "akali": "84", "akshan": "166", "alistar": "12",
	"ambessa": "799", "amumu": "32", "anivia": "34", "annie": "1", "aphelios": "523",
	"ashe": "22", "aurelionsol": "136", "aurora": "893", "azir": "268", "bard": "432",
	"belveth": "200", "blitzcrank": "53", "brand": "63", "braum": "201", "briar": "233",
	"caitlyn": "51", "camille": "164", "cassiopeia": "69", "chogath": "31", "corki": "42",
	"darius": "122", "diana": "131", "draven": "119", "drmundo": "36", "ekko": "245",
	"elise": "60", "evelynn": "28", "ezreal": "81", "fiddlesticks": "9", "fiora": "114",
	"fizz": "105", "galio": "3", "gangplank": "41", "garen": "86", "gnar": "150",
	"gragas": "79", "graves": "104", "gwen": "887", "hecarim": "120", "heimerdinger": "74",
	"hwei": "910", "illaoi": "420", "irelia": "39", "ivern": "427", "janna": "40",
	"jarvan": "59", "jax": "24", "jayce": "126", "jhin": "202", "jinx": "222",
	"kaisa": "145", "kalista": "429", "karma": "43", "karthus": "30", "kassadin": "38",
	"katarina": "55", "kayle": "10", "kayn": "141", "kennen": "85", "khazix": "121",
	"kindred": "203", "kled": "240", "kogmaw": "96", "ksante": "897", "leblanc": "7",
	"leesin": "64", "leona": "89", "lillia": "876", "lissandra": "127", "lucian": "236",
	"lulu": "117", "lux": "99", "malphite": "54", "malzahar": "90", "maokai": "57",
	"masteryi": "11", "mel": "800", "milio": "902", "missfortune": "21", "mordekaiser": "82",
	"morgana": "25", "nami": "267", "nasus": "75", "nautilus": "111", "neeko": "518",
	"nidalee": "76", "nilah": "895", "nocturne": "56", "nunu": "20", "olaf": "2",
	"orianna": "61", "ornn": "516", "pantheon": "80", "poppy": "78", "pyke": "555",
	"qiyana": "246", "quinn": "133", "rakan": "497", "rammus": "33", "reksai": "421",
	"rell": "526", "renata": "888", "renekton": "58", "rengar": "107", "riven": "92",
	"rumble": "68", "ryze": "13", "samira": "360", "sejuani": "113", "senna": "235",
	"seraphine": "147", "sett": "875", "shaco": "35", "shen": "98", "shyvana": "102",
	"singed": "27", "sion": "14", "sivir": "15", "skarner": "72", "smolder": "901",
	"sona": "37", "soraka": "16", "swain": "50", "sylas": "517", "syndra": "134",
	"tahmkench": "223", "taliyah": "163", "talon": "91", "taric": "44", "teemo": "17",
	"thresh": "412", "tristana": "18", "trundle": "48", "tryndamere": "23", "twistedfate": "4",
	"twitch": "29", "udyr": "77", "urgot": "6", "varus": "110", "vayne": "67",
	"veigar": "45", "velkoz": "161", "vex": "711", "vi": "254", "viego": "234",
	"viktor": "112", "vladimir": "8", "volibear": "106", "warwick": "19", "wukong": "62",
	"xayah": "498", "xerath": "101", "xinzhao": "5", "yasuo": "157", "yone": "777",
	"yorick": "83", "yuumi": "350", "zac": "154", "zed": "238", "zeri": "221",
	"ziggs": "115", "zilean": "26", "zoe": "142", "zyra": "143",
}

func (s *UGGScraper) GetCounters(champion, role string) (*CounterData, error) {
	// Use direct API approach - faster and more reliable
	return s.getCountersFromAPI(champion, role)
}

func (s *UGGScraper) getCountersFromAPI(champion, role string) (*CounterData, error) {
	champID, ok := championIDMap[strings.ToLower(strings.ReplaceAll(champion, " ", ""))]
	if !ok {
		return nil, fmt.Errorf("unknown champion: %s", champion)
	}

	// Direct API call
	url := fmt.Sprintf("https://stats2.u.gg/lol/1.5/matchups/16_7/ranked_solo_5x5/%s/1.5.0.json", champID)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var apiData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&apiData); err != nil {
		return nil, err
	}

	data := &CounterData{
		Champion: champion,
		Role:     role,
		Source:   s.Name(),
		Version:  "16.7",
		Tier:     "Emerald+",
	}

	// U.GG role mapping (different from standard 1-5)
	// These are internal U.GG role IDs that vary by champion
	roleMap := map[string]string{
		"top":     "17", // Gwen TOP
		"jungle":  "5",
		"mid":     "3",
		"adc":     "4",
		"support": "5",
	}

	// Determine role - use provided role or champion's default role
	effectiveRole := strings.ToLower(role)
	if effectiveRole == "" {
		// Look up champion's default role
		champKey := strings.ToLower(strings.ReplaceAll(champion, " ", ""))
		if defaultRole, ok := championDefaultRoles[champKey]; ok {
			effectiveRole = defaultRole
			data.Role = defaultRole
		} else {
			effectiveRole = "top" // Ultimate fallback
		}
	}

	roleID := roleMap[effectiveRole]
	if roleID == "" {
		roleID = "17" // Default to most common role
	}

	// Use tier 12 (Emerald+) which matches U.GG website default
	tierData, ok := apiData["12"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no tier data found")
	}

	roleData, ok := tierData[roleID].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no data for role %s", roleID)
	}

	// vsRole mapping: 1=jungle, 4=top, 5=mid (these are opponent role IDs)
	vsRoleNames := map[string]string{
		"1": "Jungle",
		"2": "ADC",  // Rare matchups
		"3": "Support", // Rare matchups
		"4": "Top",
		"5": "Mid",
	}

	// Determine which vsRole(s) to show
	// If user specified a role filter, use it. Otherwise default to standard lane matchup
	targetVsRoles := make(map[string]bool)
	requestedVsRoleName := ""
	if role != "" {
		// User explicitly requested a specific opponent role
		for vsKey, vsName := range vsRoleNames {
			if strings.EqualFold(vsName, role) {
				targetVsRoles[vsKey] = true
				requestedVsRoleName = vsName
				break
			}
		}
	}

	// Determine default matchup type
	defaultMatchupType := "Top"
	// If no specific filter, determine default vsRole based on champion's role
	if len(targetVsRoles) == 0 {
		// Standard lane matchup: top vs top, mid vs mid, etc.
		switch roleID {
		case "17", "1": // Top lane champions
			targetVsRoles["4"] = true // vs Top
			defaultMatchupType = "Top"
		case "3": // Mid lane
			targetVsRoles["5"] = true // vs Mid
			defaultMatchupType = "Mid"
		case "4": // ADC
			targetVsRoles["2"] = true // vs ADC
			defaultMatchupType = "ADC"
		case "5": // Support or Jungle
			targetVsRoles["3"] = true // vs Support (for support) or could be jungle
			defaultMatchupType = "Support"
		default:
			// Default to most common: vs Top
			targetVsRoles["4"] = true
			defaultMatchupType = "Top"
		}
	}

	// Store the matchup type for display
	if requestedVsRoleName != "" {
		data.Role = requestedVsRoleName
	} else {
		data.Role = defaultMatchupType
	}

	var matchups []Matchup
	for vsRoleKey, vsRoleWrapper := range roleData {
		// Skip if not in target vsRoles
		if !targetVsRoles[vsRoleKey] {
			continue
		}

		wrapper, ok := vsRoleWrapper.([]interface{})
		if !ok || len(wrapper) == 0 {
			continue
		}

		statsArray, ok := wrapper[0].([]interface{})
		if !ok {
			continue
		}

		vsRoleName := vsRoleNames[vsRoleKey]
		if vsRoleName == "" || vsRoleName == "?" {
			continue // Skip unknown roles
		}

		for _, stat := range statsArray {
			statArray, ok := stat.([]interface{})
			if !ok || len(statArray) < 3 {
				continue
			}

			enemyID := fmt.Sprintf("%.0f", statArray[0].(float64))
			wins, ok1 := statArray[1].(float64)
			total, ok2 := statArray[2].(float64)
			if !ok1 || !ok2 || total < 200 { // Minimum 200 games
				continue
			}

			// U.GG returns opponent's win rate (e.g., Mundo vs Gwen = 43.9%)
			// Convert to queried champion's win rate (e.g., Gwen vs Mundo = 56.1%)
			opponentWinRate := wins / total
			championWinRate := 1.0 - opponentWinRate

			enemyName := getChampionNameByID(enemyID)
			matchups = append(matchups, Matchup{
				Name:    enemyName,
				WinRate: championWinRate,  // Store champion's win rate, not opponent's
				Games:   int(total),
			})
		}
	}

	// Check if we have enough data
	if len(matchups) == 0 {
		return nil, fmt.Errorf("该分路局数太少 (vs %s 样本不足200局)", data.Role)
	}

	// Sort by win rate ascending for weak/strong
	sort.Slice(matchups, func(i, j int) bool {
		return matchups[i].WinRate < matchups[j].WinRate
	})

	// Top 5 weak (lowest win rate = hard counters)
	if len(matchups) >= 5 {
		data.WeakAgainst = matchups[:5]
	} else {
		data.WeakAgainst = matchups
	}

	// Top 5 strong (highest win rate = good matchups)
	if len(matchups) >= 5 {
		data.StrongAgainst = matchups[len(matchups)-5:]
	}

	return data, nil
}

func getChampionNameByID(id string) string {
	for name, champID := range championIDMap {
		if champID == id {
			return formatChampionName(name)
		}
	}
	return "Champion " + id
}

// RuneData represents recommended runes, items, and skills for a champion
type RuneData struct {
	Champion      string       `json:"champion"`
	Role          string       `json:"role"`
	PrimaryTree   string       `json:"primary_tree"`
	SecondaryTree string       `json:"secondary_tree"`
	Keystone      RuneInfo     `json:"keystone"`
	PrimaryRunes  []RuneInfo   `json:"primary_runes"`
	SecondaryRunes []RuneInfo `json:"secondary_runes"`
	Shards        []string     `json:"shards"`
	StartingItems []ItemInfo `json:"starting_items"`
	CoreItems     []ItemInfo `json:"core_items"`
	SkillOrder    []string     `json:"skill_order"`
	WinRate       float64      `json:"win_rate"`
	PickRate      float64      `json:"pick_rate"`
	SampleSize    int          `json:"sample_size"`
	Source        string       `json:"source"`
}

type RuneInfo struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

type ItemInfo struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

// formatChampionName converts internal names (e.g., "xinzhao") to display names ("Xin Zhao")
func formatChampionName(internal string) string {
	specialCases := map[string]string{
		"aurelionsol": "Aurelion Sol",
		"drmundo":     "Dr. Mundo",
		"jarvan":      "Jarvan IV",
		"kogmaw":      "Kog'Maw",
		"leesin":      "Lee Sin",
		"leblanc":     "LeBlanc",
		"masteryi":    "Master Yi",
		"missfortune": "Miss Fortune",
		"monkeyking":  "Wukong",
		"reksai":      "Rek'Sai",
		"tahmkench":   "Tahm Kench",
		"twistedfate": "Twisted Fate",
		"velkoz":      "Vel'Koz",
		"xinzhao":     "Xin Zhao",
		"chogath":     "Cho'Gath",
		"kaisa":       "Kai'Sa",
		"khazix":      "Kha'Zix",
		"renekton":    "Renekton",
	}
	if display, ok := specialCases[internal]; ok {
		return display
	}
	// Default: capitalize first letter only
	if len(internal) == 0 {
		return internal
	}
	return strings.ToUpper(internal[:1]) + internal[1:]
}

func (s *UGGScraper) GetMatchup(champion, enemy, role string) (*MatchupResult, error) {
	// Use the same API as GetCounters
	champID, ok := championIDMap[strings.ToLower(strings.ReplaceAll(champion, " ", ""))]
	if !ok {
		return nil, fmt.Errorf("unknown champion: %s", champion)
	}

	enemyID, ok := championIDMap[strings.ToLower(strings.ReplaceAll(enemy, " ", ""))]
	if !ok {
		return nil, fmt.Errorf("unknown enemy champion: %s", enemy)
	}

	url := fmt.Sprintf("https://stats2.u.gg/lol/1.5/matchups/16_7/ranked_solo_5x5/%s/1.5.0.json", champID)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var apiData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&apiData); err != nil {
		return nil, err
	}

	// Use same tier/role as GetCounters
	tierData, ok := apiData["12"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no tier data found")
	}

	roleMap := map[string]string{"top": "17", "jungle": "5", "mid": "3", "adc": "4", "support": "5"}
	roleID := roleMap[strings.ToLower(role)]
	if roleID == "" {
		roleID = "17"
	}

	roleData, ok := tierData[roleID].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no data for role")
	}

	// Check all vsRoles to find the enemy
	for _, vsKey := range []string{"1", "2", "3", "4", "5"} {
		vsRoleWrapper, ok := roleData[vsKey].([]interface{})
		if !ok || len(vsRoleWrapper) == 0 {
			continue
		}

		statsArray, ok := vsRoleWrapper[0].([]interface{})
		if !ok {
			continue
		}

		// Look for enemy in this array
		for _, stat := range statsArray {
			statArray, ok := stat.([]interface{})
			if !ok || len(statArray) < 3 {
				continue
			}

			eid := fmt.Sprintf("%.0f", statArray[0].(float64))
			if eid == enemyID {
				wins := statArray[1].(float64)
				total := statArray[2].(float64)
				if total >= 200 {
					return &MatchupResult{
						Champion:   champion,
						Enemy:      enemy,
						WinRate:    wins / total,
						SampleSize: int(total),
						Source:     "ugg",
					}, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("matchup not found")
}

// GetRunes fetches recommended runes for a champion from U.GG
func (s *UGGScraper) GetRunes(champion, role string) (*RuneData, error) {
	champID, ok := championIDMap[strings.ToLower(strings.ReplaceAll(champion, " ", ""))]
	if !ok {
		return nil, fmt.Errorf("unknown champion: %s", champion)
	}

	// U.GG overview API contains rune data
	url := fmt.Sprintf("https://stats2.u.gg/lol/1.5/overview/16_7/ranked_solo_5x5/%s/1.5.0.json", champID)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var apiData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&apiData); err != nil {
		return nil, err
	}

	// Parse rune data from API response
	return s.parseRunesFromData(apiData, champion, role)
}

// Champion default roles mapping (each champion assigned to their primary/most common role)
var championDefaultRoles = map[string]string{
	// Top (62)
	"aatrox": "top", "ambessa": "top", "camille": "top", "cho": "top", "darius": "top",
	"drmundo": "top", "fiora": "top", "gangplank": "top", "garen": "top", "gnar": "top",
	"gragas": "top", "gwen": "top", "heimerdinger": "top", "illaoi": "top", "irelia": "top",
	"jax": "top", "jayce": "top", "kayle": "top", "kennen": "top", "kled": "top",
	"malphite": "top", "maokai": "top", "mordekaiser": "top", "nasus": "top", "olaf": "top",
	"ornn": "top", "poppy": "top", "quinn": "top", "renekton": "top", "riven": "top",
	"rumble": "top", "sett": "top", "shen": "top", "sion": "top", "skarner": "top",
	"teemo": "top", "trundle": "top", "tryndamere": "top", "urgot": "top", "volibear": "top",
	"warwick": "top", "wukong": "top", "yasuo": "top", "yone": "top", "yorick": "top",
	// Jungle (48)
	"amumu": "jungle", "belveth": "jungle", "brand": "jungle", "briar": "jungle", "diana": "jungle",
	"ekko": "jungle", "elise": "jungle", "evelynn": "jungle", "fiddlesticks": "jungle", "graves": "jungle",
	"hecarim": "jungle", "ivern": "jungle", "jarvan": "jungle", "karthus": "jungle", "kayn": "jungle",
	"khazix": "jungle", "kindred": "jungle", "leesin": "jungle", "lillia": "jungle", "masteryi": "jungle",
	"nidalee": "jungle", "nocturne": "jungle", "nunu": "jungle", "pantheon": "jungle", "rammus": "jungle",
	"reksai": "jungle", "rell": "jungle", "rengar": "jungle", "sejuani": "jungle", "shaco": "jungle",
	"shyvana": "jungle", "taliyah": "jungle", "udyr": "jungle", "vi": "jungle", "viego": "jungle",
	"xinzhao": "jungle", "zac": "jungle",
	// Mid (56)
	"ahri": "mid", "akali": "mid", "akshan": "mid", "anivia": "mid", "annie": "mid",
	"aurelionsol": "mid", "aurora": "mid", "azir": "mid", "cassiopeia": "mid", "corki": "mid",
	"fizz": "mid", "galio": "mid", "hwei": "mid", "kassadin": "mid", "katarina": "mid",
	"leblanc": "mid", "lissandra": "mid", "lux": "mid", "malzahar": "mid", "mel": "mid",
	"naafiri": "mid", "neeko": "mid", "orianna": "mid", "qiyana": "mid", "ryze": "mid",
	"swain": "mid", "sylas": "mid", "syndra": "mid", "talon": "mid", "tristana": "mid",
	"twistedfate": "mid", "veigar": "mid", "velkoz": "mid", "vex": "mid", "viktor": "mid",
	"vladimir": "mid", "xerath": "mid", "zed": "mid", "ziggs": "mid", "zoe": "mid",
	// ADC (24)
	"aphelios": "adc", "ashe": "adc", "caitlyn": "adc", "draven": "adc", "ezreal": "adc",
	"jhin": "adc", "jinx": "adc", "kaisa": "adc", "kalista": "adc", "kogmaw": "adc",
	"lucian": "adc", "missfortune": "adc", "nilah": "adc", "samira": "adc", "senna": "adc",
	"seraphine": "adc", "sivir": "adc", "smolder": "adc", "twitch": "adc", "varus": "adc",
	"vayne": "adc", "xayah": "adc", "zeri": "adc",
	// Support (39)
	"alistar": "support", "bard": "support", "blitzcrank": "support", "braum": "support",
	"janna": "support", "karma": "support", "leona": "support", "lulu": "support", "milio": "support",
	"morgana": "support", "nautilus": "support", "pyke": "support", "rakan": "support",
	"renataglasc": "support", "sona": "support", "soraka": "support", "tahmkench": "support",
	"taric": "support", "thresh": "support", "yuumi": "support", "zyra": "support",
}

// Rune tree ID mapping
var runeTreeNames = map[int]string{
	8000: "精密",
	8100: "主宰",
	8200: "巫术",
	8300: "启迪",
	8400: "坚决",
}

// Rune ID to name mapping
var runeNames = map[int]string{
	// Precision 精密 (8000)
	8005: "强攻", 8008: "致命节奏", 8021: "迅捷步法", 8010: "征服者",
	9101: "过量治疗", 9111: "凯旋", 8009: "气定神闲",
	8299: "传说:血统",
	9104: "传说:欢欣", 9105: "传说:韧性",
	8014: "致命一击", 8015: "砍倒", 8016: "坚毅不倒",
	// Domination 主宰 (8100)
	8112: "电刑", 8124: "掠食者", 8128: "黑暗收割", 9923: "丛刃",
	8126: "恶意中伤", 8139: "血之滋味", 8140: "猛然冲击",
	8138: "眼球收集器", 8300: "幽灵魄罗", 8301: "僵尸守卫",
	8135: "寻宝猎人", 8134: "贪欲猎手",
	// Sorcery 巫术 (8200)
	8214: "召唤:艾黎", 8229: "奥术彗星", 8230: "相位猛冲",
	8224: "无效化之法球", 8226: "法力流系带", 8275: "灵光披风",
	8210: "绝对专注", 8234: "焦灼", 8233: "风暴聚集",
	// Resolve 坚决 (8400)
	8437: "不灭之握", 8439: "余震", 8465: "守护者",
	8446: "爆破", 8473: "生命之泉", 8451: "调节", 8453: "复苏之风", 8401: "骸骨镀层",
	8444: "复苏", 8472: "坚定",
	// Inspiration 启迪 (8300)
	8351: "冰川增幅", 8360: "启封的秘籍", 8369: "先攻",
	8306: "海克斯科技闪现罗网", 8304: "神奇之鞋", 8313: "未来市场", 8321: "饼干配送",
	8345: "星界洞悉", 8347: "宇宙洞悉", 8410: "迅捷",
	// Stats Shards (格式为字符串)
	5005: "+10%攻速", 5007: "+10%冷却", 5008: "+9适应之力",
	5001: "+15-90生命", 5002: "+6护甲", 5003: "+8魔抗",
	// Missing runes
	8017: "切割", 8105: "赏金猎人",
}

// Common item names mapping
var itemNames = map[int]string{
	// Starting items
	1054: "多兰之盾", 1055: "多兰之刃", 1056: "多兰之戒",
	2003: "生命药水", 2031: "腐蚀药水", 2033: "复用型药水",
	2055: "控制守卫", 1101: "冰雹刀刃", 1102: "灰烬小刀",
	// Boots
	3006: "狂战士胫甲", 3009: "轻灵之靴", 3020: "法师之靴",
	3047: "忍者足具", 3111: "水银之靴", 3117: "疾行之靴",
	3158: "明朗之靴", 3005: "铁板靴",
	// Mythic/Legendary items (common)
	2510: "裂隙制造者",
	3078: "三相之力", 6632: "神圣分离者", 6662: "冰霜护手",
	3153: "破败王者之刃", 3142: "幽梦之灵", 6691: "德拉克斯的暮刃",
	6692: "星蚀", 6676: "海妖杀手", 6673: "不朽盾弓",
	3074: "贪欲九头蛇", 3748: "巨型九头蛇", 6333: "死亡之舞",
	3026: "守护天使", 3033: "凡性的提醒", 3036: "领主的致意",
	3046: "幻影之舞", 3085: "疾射火炮", 3094: "疾射火炮",
	3115: "纳什之牙", 3100: "巫妖之祸", 3157: "中娅沙漏",
	3165: "莫雷洛秘典", 3135: "虚空之杖", 3089: "灭世者的死亡之帽",
	3116: "瑞莱的冰晶节杖", 3001: "深渊面具", 3065: "振奋盔甲",
	3075: "荆棘之甲", 3143: "兰顿之兆", 3068: "日炎圣盾",
	3193: "石像鬼石板甲", 3109: "骑士之誓", 3050: "天顶锋刃",
	3084: "狂徒铠甲", 6617: "流水法杖", 4629: "炼金科技纯化器",
	4645: "视界专注", 3003: "大天使之杖", 3027: "时光之杖",
	3031: "无尽之刃", 3032: "疾射火炮", 3035: "最后的轻语",
	3053: "斯特拉克的挑战护手", 3066: "黑色切割者", 3071: "黑色切割者",
	3072: "饮血剑", 3077: "提亚马特", 3083: "狂徒铠甲",
	3087: "斯塔缇克电刃", 3091: "智慧末刃", 3095: "疾射火炮",
	3102: "女妖面纱", 3107: "救赎", 3110: "冰霜之心",
	3112: "冰霜之牙", 3113: "艾希的霜火护手", 3114: "炽热香炉",
	3119: "寒冬之握", 3124: "血色之刃", 3128: "灭世者的死亡之帽",
	3133: "考尔菲德的战锤", 3134: "锯齿短匕", 3139: "水银弯刀",
	3140: "水银饰带", 3144: "吸血鬼节杖", 3145: "海克斯科技枪",
	3146: "海克斯科技枪刃", 3147: "德拉克萨的暮刃", 3152: "海克斯科技火箭腰带",
	3155: "饮魔刀", 3156: "玛莫提乌斯之噬", 3161: "破舰者",
	3177: "死刑宣告", 3179: "暗影阔剑", 3181: "收集者",
	3184: "血色之刃", 3190: "钢铁烈阳之匣", 4005: "月石再生器",
	4401: "自然之力", 4403: "灭世者的死亡之帽", 4628: "炼金科技纯化器",
	4630: "流水法杖", 4633: "中娅沙漏", 4636: "暗夜收割者",
	4637: "恶魔之拥", 4638: "视界专注", 4643: "星界驱驰",
	4644: "影焰", 4646: "虚空之杖", 6035: "巨蛇之牙",
	6660: "日炎圣盾", 6664: "涡轮炼金罐",
	6665: "炼金科技纯化器", 6667: "石像鬼石板甲", 6670: "海妖杀手",
	6671: "不朽盾弓", 6672: "狂风之力", 6675: "暗夜收割者",
	6693: "德拉克萨的暮刃", 6694: "星蚀", 6695: "赛瑞尔达的怨恨",
	6696: "梦想海域", 7000: "灭世者的死亡之帽",
}

func (s *UGGScraper) parseRunesFromData(data map[string]interface{}, champion, role string) (*RuneData, error) {
	runeData := &RuneData{
		Champion: champion,
		Role:     role,
		Source:   "ugg",
	}

	// U.GG overview API role mapping
	// For overview API, we need to find the correct U.GG role ID for each lane
	// This varies by champion, so we look for the most common one
	roleKey := "4" // Default top for most champions
	if role != "" {
		switch strings.ToLower(role) {
		case "top":
			roleKey = "4"
		case "jungle", "jg":
			roleKey = "1"
		case "mid":
			roleKey = "5"
		case "adc", "bot":
			roleKey = "2"
		case "support", "sup":
			roleKey = "3"
		}
	}

	// Extract tier 1 (all ranks) or tier 12 (Emerald+)
	tier, ok := data["1"].(map[string]interface{})
	if !ok {
		// Try tier 12 (Emerald+)
		tier, ok = data["12"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("no rune data found")
		}
	}

	// Get role data - it's a map of vsRole -> data
	roleData, ok := tier[roleKey].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no rune data for role")
	}

	// Find the first vsRole that has data
	var runeBuildWrapper []interface{}
	for _, vsRoleKey := range []string{"1", "2", "3", "4", "5"} {
		if wrapper, ok := roleData[vsRoleKey].([]interface{}); ok && len(wrapper) > 0 {
			runeBuildWrapper = wrapper
			break
		}
	}
	if runeBuildWrapper == nil {
		return nil, fmt.Errorf("no rune build data found")
	}

	// runeBuildWrapper is a list of builds, each build is an array
	// Find the build with most games (sample size)
	var bestBuild []interface{}
	maxGames := 0.0

	for _, item := range runeBuildWrapper {
		buildData, ok := item.([]interface{})
		if !ok || len(buildData) < 5 {
			continue
		}
		// First element of build is [games, wins, ...]
		firstElem, ok := buildData[0].([]interface{})
		if !ok || len(firstElem) < 2 {
			continue
		}
		// First two values: [0]=games, [1]=wins
		if games, ok := firstElem[0].(float64); ok && games > maxGames {
			maxGames = games
			bestBuild = buildData
		}
	}

	if bestBuild == nil {
		return nil, fmt.Errorf("no valid rune build found")
	}

	// Use the best build (most games)
	// bestBuild = [[games, wins, primary, secondary, [runes...]], [item1], [item2], [skill], ...]

	// Extract rune data from index 0
	runeDataArray, ok := bestBuild[0].([]interface{})
	if !ok || len(runeDataArray) < 5 {
		return nil, fmt.Errorf("invalid rune data structure")
	}

	// Extract win/pick stats - U.GG uses [games, wins, ...]
	if games, ok := runeDataArray[0].(float64); ok {
		if wins, ok := runeDataArray[1].(float64); ok && games > 0 {
			runeData.WinRate = wins / games
			runeData.SampleSize = int(games)
		}
	}

	// Extract rune tree IDs
	if primaryTreeID, ok := runeDataArray[2].(float64); ok {
		runeData.PrimaryTree = runeTreeNames[int(primaryTreeID)]
	}
	if secondaryTreeID, ok := runeDataArray[3].(float64); ok {
		runeData.SecondaryTree = runeTreeNames[int(secondaryTreeID)]
	}

	// Extract rune array
	// Structure: [keystone, primary1, primary2, primary3, secondary1, secondary2, ...]
	if runeArray, ok := runeDataArray[4].([]interface{}); ok && len(runeArray) >= 4 {
		// First rune is keystone
		if keystoneID, ok := runeArray[0].(float64); ok {
			runeData.Keystone = RuneInfo{
				Name: runeNames[int(keystoneID)],
				ID:   int(keystoneID),
			}
		}
		// Next 3 are primary tree runes
		for i := 1; i < 4 && i < len(runeArray); i++ {
			if runeID, ok := runeArray[i].(float64); ok {
				runeData.PrimaryRunes = append(runeData.PrimaryRunes, RuneInfo{
					Name: runeNames[int(runeID)],
					ID:   int(runeID),
				})
			}
		}
		// Remaining are secondary tree runes (usually 2)
		for i := 4; i < len(runeArray); i++ {
			if runeID, ok := runeArray[i].(float64); ok {
				runeData.SecondaryRunes = append(runeData.SecondaryRunes, RuneInfo{
					Name: runeNames[int(runeID)],
					ID:   int(runeID),
				})
			}
		}
	}

	// Extract starting items from index 2
	// Structure: [wins, games, [itemIDs...]]
	if len(bestBuild) > 2 {
		if itemData, ok := bestBuild[2].([]interface{}); ok && len(itemData) >= 3 {
			if itemIDs, ok := itemData[2].([]interface{}); ok {
				for _, id := range itemIDs {
					if itemID, ok := id.(float64); ok {
						runeData.StartingItems = append(runeData.StartingItems, ItemInfo{
							Name: itemNames[int(itemID)],
							ID:   int(itemID),
						})
					}
				}
			}
		}
	}

	// Extract core items from index 3
	if len(bestBuild) > 3 {
		if itemData, ok := bestBuild[3].([]interface{}); ok && len(itemData) >= 3 {
			if itemIDs, ok := itemData[2].([]interface{}); ok {
				for _, id := range itemIDs {
					if itemID, ok := id.(float64); ok {
						runeData.CoreItems = append(runeData.CoreItems, ItemInfo{
							Name: itemNames[int(itemID)],
							ID:   int(itemID),
						})
					}
				}
			}
		}
	}

	// Extract skill order from index 4
	if len(bestBuild) > 4 {
		if skillData, ok := bestBuild[4].([]interface{}); ok && len(skillData) >= 3 {
			if skills, ok := skillData[2].([]interface{}); ok {
				for _, s := range skills {
					if skill, ok := s.(string); ok {
						runeData.SkillOrder = append(runeData.SkillOrder, skill)
					}
				}
			}
		}
	}

	// Extract shards from index 8 (if available)
	// Shards are numeric IDs like 5008, need to convert to int and look up name
	if len(bestBuild) > 8 {
		if shardData, ok := bestBuild[8].([]interface{}); ok && len(shardData) >= 3 {
			if shards, ok := shardData[2].([]interface{}); ok {
				for _, s := range shards {
					// Shards can be float64 (from JSON numbers) or string
					var shardID int
					if f, ok := s.(float64); ok {
						shardID = int(f)
					} else if str, ok := s.(string); ok {
						shardID, _ = strconv.Atoi(str)
					}
					if shardID > 0 {
						if name, ok := runeNames[shardID]; ok {
							runeData.Shards = append(runeData.Shards, name)
						} else {
							runeData.Shards = append(runeData.Shards, fmt.Sprintf("[%d]", shardID))
						}
					}
				}
			}
		}
	}

	return runeData, nil
}

func extractMatchupFromUGGPage(doc *goquery.Document, champion, enemy string) (*MatchupResult, error) {
	// Look for win rate display
	var winRate float64
	var found bool

	// U.GG typically shows the win rate in a prominent element
	doc.Find("[class*='win-rate'], [class*='winrate'], .stat-value").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if strings.Contains(text, "%") {
			wr := parseWinRate(text)
			if wr > 0 {
				winRate = wr
				found = true
			}
		}
	})

	if !found {
		return nil, fmt.Errorf("win rate not found in page")
	}

	return &MatchupResult{
		Champion: champion,
		Enemy:    enemy,
		WinRate:  winRate,
		Source:   "ugg",
	}, nil
}

func parseUGGSSRData(scriptText, champion, enemy string) (*MatchupResult, error) {
	// Extract JSON from window.__SSR_DATA__ = {...};
	start := strings.Index(scriptText, "window.__SSR_DATA__ = ")
	if start == -1 {
		return nil, fmt.Errorf("SSR_DATA not found")
	}
	start += len("window.__SSR_DATA__ = ")

	// Find the end of the JSON (semicolon)
	end := strings.LastIndex(scriptText[start:], "};")
	if end == -1 {
		end = len(scriptText) - start
	} else {
		end += 1 // Include the closing brace
	}

	jsonStr := scriptText[start : start+end]

	// For now, try to extract win rate via regex
	// Look for patterns like "winRate":51.23 or "win_rate":"51.23%"
	wrPattern := `"win_?rate"[:\s]*"?([0-9.]+)"?`
	re := regexp.MustCompile(`(?i)` + wrPattern)
	matches := re.FindStringSubmatch(jsonStr)

	if len(matches) > 1 {
		wr, _ := strconv.ParseFloat(matches[1], 64)
		if wr > 1 {
			wr = wr / 100
		}
		return &MatchupResult{
			Champion: champion,
			Enemy:    enemy,
			WinRate:  wr,
			Source:   "ugg",
		}, nil
	}

	return nil, fmt.Errorf("win rate not found in SSR data")
}

// MultiScraper tries multiple sources
type MultiScraper struct {
	scrapers        []Scraper
	matchupScrapers []MatchupScraper
	cache           *ScraperCache
}

func NewMultiScraper() *MultiScraper {
	// UGGScraper uses direct API, more reliable
	scrapers := []Scraper{
		NewUGGScraper(),
		NewLeagueOfGraphsScraper(),
	}

	matchupScrapers := []MatchupScraper{
		NewUGGScraper(),
	}

	// Add OPGG scraper if Playwright is available
	if opgg := initOPGG(); opgg != nil {
		if ms, ok := opgg.(MatchupScraper); ok {
			matchupScrapers = append(matchupScrapers, ms)
		}
		// Don't add OPGG to scrapers since it's blocked by Cloudflare
	}

	return &MultiScraper{
		scrapers:        scrapers,
		matchupScrapers: matchupScrapers,
		cache:           NewScraperCache(5 * time.Minute), // Cache for 5 minutes
	}
}

func (m *MultiScraper) GetCounters(champion, role string) (*CounterData, error) {
	var lastErr error
	for _, s := range m.scrapers {
		// Check cache first
		cacheKey := makeCacheKey(s.Name(), champion, role)
		if cached, found := m.cache.Get(cacheKey); found {
			return cached, nil
		}

		data, err := s.GetCounters(champion, role)
		if err == nil {
			// Store in cache
			m.cache.Set(cacheKey, data)
			return data, nil
		}
		lastErr = fmt.Errorf("%s: %w", s.Name(), err)
	}
	return nil, lastErr
}

func (m *MultiScraper) GetMatchup(champion, enemy, role string) (*MatchupResult, error) {
	var lastErr error
	for _, s := range m.matchupScrapers {
		// Check cache first
		cacheKey := makeMatchupCacheKey(s.Name(), champion, enemy, role)
		if cached, found := m.cache.GetMatchup(cacheKey); found {
			return cached, nil
		}

		data, err := s.GetMatchup(champion, enemy, role)
		if err == nil {
			// Store in cache
			m.cache.SetMatchup(cacheKey, data)
			return data, nil
		}
		lastErr = fmt.Errorf("%s: %w", s.Name(), err)
	}
	return nil, lastErr
}

func keysOf(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
