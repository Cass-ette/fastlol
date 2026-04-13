package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// RiotClient handles Riot Games API requests
type RiotClient struct {
	apiKey     string
	httpClient *http.Client
}

// Riot regions and their routing values
var RegionRouting = map[string]string{
	"br1":   "americas",
	"eun1":  "europe",
	"euw1":  "europe",
	"jp1":   "asia",
	"kr":    "asia",
	"la1":   "americas",
	"la2":   "americas",
	"na1":   "americas",
	"oc1":   "sea",
	"ph2":   "sea",
	"ru":    "europe",
	"sg2":   "sea",
	"th2":   "sea",
	"tr1":   "europe",
	"tw2":   "sea",
	"vn2":   "sea",
}

// NewRiotClient creates a new Riot API client
func NewRiotClient(apiKey string) *RiotClient {
	return &RiotClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *RiotClient) doRequest(baseURL, path string) ([]byte, error) {
	req, err := http.NewRequest("GET", baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Riot-Token", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (c *RiotClient) getRegionalBaseURL(region string) string {
	routing, ok := RegionRouting[region]
	if !ok {
		routing = "americas"
	}
	return fmt.Sprintf("https://%s.api.riotgames.com", routing)
}

func (c *RiotClient) getPlatformBaseURL(region string) string {
	return fmt.Sprintf("https://%s.api.riotgames.com", region)
}

// Summoner represents a summoner's basic info
type Summoner struct {
	ID            string `json:"id"`
	AccountID     string `json:"accountId"`
	PUUID         string `json:"puuid"`
	Name          string `json:"name"`
	ProfileIconID int    `json:"profileIconId"`
	RevisionDate  int64  `json:"revisionDate"`
	SummonerLevel int    `json:"summonerLevel"`
}

// RankedInfo represents ranked queue stats
type RankedInfo struct {
	QueueType    string `json:"queueType"`
	Tier         string `json:"tier"`
	Rank         string `json:"rank"`
	LeaguePoints int    `json:"leaguePoints"`
	Wins         int    `json:"wins"`
	Losses       int    `json:"losses"`
	HotStreak    bool   `json:"hotStreak"`
}

// MatchMetadata represents basic match info
type MatchMetadata struct {
	MatchID       string `json:"matchId"`
	GameCreation  int64  `json:"gameCreation"`
	GameDuration  int    `json:"gameDuration"`
	GameMode      string `json:"gameMode"`
	GameType      string `json:"gameType"`
	Win           bool   `json:"win"`
	ChampionName  string `json:"championName"`
	Kills         int    `json:"kills"`
	Deaths        int    `json:"deaths"`
	Assists       int    `json:"assists"`
}

// FormatDuration converts seconds to mm:ss format
func FormatDuration(seconds int) string {
	m := seconds / 60
	s := seconds % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

// WinRate calculates win rate percentage
func WinRate(wins, losses int) float64 {
	total := wins + losses
	if total == 0 {
		return 0
	}
	return float64(wins) / float64(total) * 100
}

// GetSummonerByName fetches summoner info by name (deprecated in API v5 but still works for some regions)
func (c *RiotClient) GetSummonerByName(region, name string) (*Summoner, error) {
	encodedName := url.QueryEscape(name)
	baseURL := c.getPlatformBaseURL(region)
	data, err := c.doRequest(baseURL, "/lol/summoner/v4/summoners/by-name/"+encodedName)
	if err != nil {
		return nil, err
	}

	var summoner Summoner
	if err := json.Unmarshal(data, &summoner); err != nil {
		return nil, fmt.Errorf("parse summoner failed: %w", err)
	}
	return &summoner, nil
}

// GetSummonerByTag fetches summoner by gameName and tagLine (new Riot ID system)
func (c *RiotClient) GetSummonerByTag(region, gameName, tagLine string) (*Summoner, error) {
	encodedName := url.QueryEscape(gameName)
	baseURL := c.getPlatformBaseURL(region)
	path := fmt.Sprintf("/riot/account/v1/accounts/by-riot-id/%s/%s", encodedName, tagLine)
	data, err := c.doRequest(baseURL, path)
	if err != nil {
		return nil, err
	}

	// This returns puuid, then we need to get summoner by puuid
	var account struct {
		PUUID    string `json:"puuid"`
		GameName string `json:"gameName"`
		TagLine  string `json:"tagLine"`
	}
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, fmt.Errorf("parse account failed: %w", err)
	}

	return c.GetSummonerByPUUID(region, account.PUUID)
}

// GetSummonerByPUUID fetches summoner by PUUID
func (c *RiotClient) GetSummonerByPUUID(region, puuid string) (*Summoner, error) {
	baseURL := c.getPlatformBaseURL(region)
	data, err := c.doRequest(baseURL, "/lol/summoner/v4/summoners/by-puuid/"+puuid)
	if err != nil {
		return nil, err
	}

	var summoner Summoner
	if err := json.Unmarshal(data, &summoner); err != nil {
		return nil, fmt.Errorf("parse summoner failed: %w", err)
	}

	// Copy game name from Riot ID if available
	if summoner.Name == "" {
		summoner.Name = "Summoner"
	}

	return &summoner, nil
}

// GetRankedStats fetches ranked stats for a summoner
func (c *RiotClient) GetRankedStats(region, summonerID string) ([]RankedInfo, error) {
	baseURL := c.getPlatformBaseURL(region)
	data, err := c.doRequest(baseURL, "/lol/league/v4/entries/by-summoner/"+summonerID)
	if err != nil {
		return nil, err
	}

	var stats []RankedInfo
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, fmt.Errorf("parse ranked stats failed: %w", err)
	}
	return stats, nil
}

// GetRecentMatches fetches recent match IDs for a player
func (c *RiotClient) GetRecentMatches(region, puuid string, count int) ([]string, error) {
	if count <= 0 {
		count = 10
	}
	if count > 20 {
		count = 20
	}

	baseURL := c.getRegionalBaseURL(region)
	path := fmt.Sprintf("/lol/match/v5/matches/by-puuid/%s/ids?start=0&count=%d", puuid, count)
	data, err := c.doRequest(baseURL, path)
	if err != nil {
		return nil, err
	}

	var matches []string
	if err := json.Unmarshal(data, &matches); err != nil {
		return nil, fmt.Errorf("parse matches failed: %w", err)
	}
	return matches, nil
}

// GetMatchInfo fetches detailed match info
func (c *RiotClient) GetMatchInfo(region, matchID string) (*MatchInfo, error) {
	baseURL := c.getRegionalBaseURL(region)
	data, err := c.doRequest(baseURL, "/lol/match/v5/matches/"+matchID)
	if err != nil {
		return nil, err
	}

	var info MatchInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parse match failed: %w", err)
	}
	return &info, nil
}

// MatchInfo represents full match data
type MatchInfo struct {
	Metadata struct {
		MatchID string `json:"matchId"`
	} `json:"metadata"`
	Info struct {
		GameCreation int64  `json:"gameCreation"`
		GameDuration int    `json:"gameDuration"`
		GameMode     string `json:"gameMode"`
		GameType     string `json:"gameType"`
		Participants []struct {
			PUUID        string `json:"puuid"`
			ChampionName string `json:"championName"`
			Kills        int    `json:"kills"`
			Deaths       int    `json:"deaths"`
			Assists      int    `json:"assists"`
			Win          bool   `json:"win"`
			Role         string `json:"role"`
			Lane         string `json:"lane"`
		} `json:"participants"`
	} `json:"info"`
}

// GetPlayerMatchMetadata extracts match metadata for a specific player
func (m *MatchInfo) GetPlayerMatchMetadata(puuid string) *MatchMetadata {
	for _, p := range m.Info.Participants {
		if p.PUUID == puuid {
			return &MatchMetadata{
				MatchID:      m.Metadata.MatchID,
				GameCreation: m.Info.GameCreation,
				GameDuration: m.Info.GameDuration,
				GameMode:     m.Info.GameMode,
				GameType:     m.Info.GameType,
				Win:          p.Win,
				ChampionName: p.ChampionName,
				Kills:        p.Kills,
				Deaths:       p.Deaths,
				Assists:      p.Assists,
			}
		}
	}
	return nil
}
