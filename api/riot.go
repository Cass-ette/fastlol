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
	MatchID             string `json:"matchId"`
	GameCreation        int64  `json:"gameCreation"`
	GameDuration        int    `json:"gameDuration"`
	GameMode            string `json:"gameMode"`
	GameType            string `json:"gameType"`
	Win                 bool   `json:"win"`
	ChampionName        string `json:"championName"`
	Kills               int    `json:"kills"`
	Deaths              int    `json:"deaths"`
	Assists             int    `json:"assists"`
	TotalMinionsKilled  int    `json:"totalMinionsKilled"`
	NeutralMinionsKilled int    `json:"neutralMinionsKilled"`
}

// ChampionMastery represents champion mastery data
type ChampionMastery struct {
	PUUID                   string `json:"puuid"`
	ChampionID              int64  `json:"championId"`
	ChampionLevel           int    `json:"championLevel"`
	ChampionPoints          int    `json:"championPoints"`
	LastPlayTime            int64  `json:"lastPlayTime"`
	ChampionPointsSinceLastLevel int64 `json:"championPointsSinceLastLevel"`
	ChampionPointsUntilNextLevel int64 `json:"championPointsUntilNextLevel"`
	TokensEarned            int    `json:"tokensEarned"`
	ChestGranted            bool   `json:"chestGranted"`
}

// Account represents basic account info from account-v1
type Account struct {
	PUUID    string `json:"puuid"`
	GameName string `json:"gameName"`
	TagLine  string `json:"tagLine"`
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
	encodedName := url.PathEscape(name)
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

// GetAccountByTag fetches account by Riot ID (gameName#tagLine)
// This is the most reliable endpoint for development keys
func (c *RiotClient) GetAccountByTag(region, gameName, tagLine string) (*Account, error) {
	encodedName := url.PathEscape(gameName)
	encodedTag := url.PathEscape(tagLine)
	// account-v1 uses regional routing (americas/europe/asia), not platform routing
	baseURL := c.getRegionalBaseURL(region)
	path := fmt.Sprintf("/riot/account/v1/accounts/by-riot-id/%s/%s", encodedName, encodedTag)
	data, err := c.doRequest(baseURL, path)
	if err != nil {
		return nil, err
	}

	var account Account
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, fmt.Errorf("parse account failed: %w", err)
	}
	return &account, nil
}

// GetSummonerByTag fetches summoner by gameName and tagLine (new Riot ID system)
// Note: This may fail with 403 for development keys on summoner-v4 endpoint
func (c *RiotClient) GetSummonerByTag(region, gameName, tagLine string) (*Summoner, error) {
	// Get account first
	account, err := c.GetAccountByTag(region, gameName, tagLine)
	if err != nil {
		return nil, err
	}

	// Try to get summoner data (may fail with 403 for dev keys)
	summoner, err := c.GetSummonerByPUUID(region, account.PUUID)
	if err != nil {
		// If summoner-v4 fails, return minimal info from account
		return &Summoner{
			PUUID:         account.PUUID,
			Name:          fmt.Sprintf("%s#%s", account.GameName, account.TagLine),
			SummonerLevel: 0, // Unknown due to API restriction
		}, nil
	}
	return summoner, nil
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

// GetChampionMastery fetches top champion mastery entries for a player
// Uses platform routing (kr, na1, etc.) - available for development keys
func (c *RiotClient) GetChampionMastery(region, puuid string, count int) ([]ChampionMastery, error) {
	if count <= 0 {
		count = 3
	}
	if count > 10 {
		count = 10
	}

	baseURL := c.getPlatformBaseURL(region)
	path := fmt.Sprintf("/lol/champion-mastery/v4/champion-masteries/by-puuid/%s/top?count=%d", puuid, count)
	data, err := c.doRequest(baseURL, path)
	if err != nil {
		return nil, err
	}

	var masteries []ChampionMastery
	if err := json.Unmarshal(data, &masteries); err != nil {
		return nil, fmt.Errorf("parse champion mastery failed: %w", err)
	}
	return masteries, nil
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
			PUUID                 string `json:"puuid"`
			ChampionName          string `json:"championName"`
			Kills                 int    `json:"kills"`
			Deaths                int    `json:"deaths"`
			Assists               int    `json:"assists"`
			Win                   bool   `json:"win"`
			Role                  string `json:"role"`
			Lane                  string `json:"lane"`
			TotalMinionsKilled   int    `json:"totalMinionsKilled"`
			NeutralMinionsKilled int    `json:"neutralMinionsKilled"`
		} `json:"participants"`
	} `json:"info"`
}

// GetPlayerMatchMetadata extracts match metadata for a specific player
func (m *MatchInfo) GetPlayerMatchMetadata(puuid string) *MatchMetadata {
	for _, p := range m.Info.Participants {
		if p.PUUID == puuid {
			return &MatchMetadata{
				MatchID:              m.Metadata.MatchID,
				GameCreation:          m.Info.GameCreation,
				GameDuration:          m.Info.GameDuration,
				GameMode:              m.Info.GameMode,
				GameType:              m.Info.GameType,
				Win:                   p.Win,
				ChampionName:          p.ChampionName,
				Kills:                 p.Kills,
				Deaths:                p.Deaths,
				Assists:               p.Assists,
				TotalMinionsKilled:    p.TotalMinionsKilled,
				NeutralMinionsKilled:  p.NeutralMinionsKilled,
			}
		}
	}
	return nil
}

// MatchParticipant represents a single participant with full identity info
type MatchParticipant struct {
	RiotIDGameName string `json:"riotIdGameName"`
	RiotIDTagline  string `json:"riotIdTagline"`
	ChampionName   string `json:"championName"`
	ChampionID     int    `json:"championId"`
	TeamID         int    `json:"teamId"`
	Win            bool   `json:"win"`
	Kills          int    `json:"kills"`
	Deaths         int    `json:"deaths"`
	Assists        int    `json:"assists"`
	Role           string `json:"role"`
	Lane            string `json:"lane"`
}

// FullMatchInfo is the full match detail with all participants
type FullMatchInfo struct {
	MatchID       string
	GameCreation  int64
	GameDuration  int
	GameMode      string
	Participants  []MatchParticipant
}

// GetFullMatchInfo returns match with all participant details
func (c *RiotClient) GetFullMatchInfo(region, matchID string) (*FullMatchInfo, error) {
	data, err := c.doRequest(c.getRegionalBaseURL(region), "/lol/match/v5/matches/"+matchID)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Metadata struct {
			MatchID string `json:"matchId"`
		} `json:"metadata"`
		Info struct {
			GameCreation int64  `json:"gameCreation"`
			GameDuration int    `json:"gameDuration"`
			GameMode     string `json:"gameMode"`
			Participants []struct {
				RiotIDGameName string `json:"riotIdGameName"`
				RiotIDTagline  string `json:"riotIdTagline"`
				ChampionName   string `json:"championName"`
				ChampionID     int    `json:"championId"`
				TeamID         int    `json:"teamId"`
				Win            bool   `json:"win"`
				Kills          int    `json:"kills"`
				Deaths         int    `json:"deaths"`
				Assists        int    `json:"assists"`
				Role           string `json:"role"`
				Lane            string `json:"lane"`
			} `json:"participants"`
		} `json:"info"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse full match failed: %w", err)
	}

	result := &FullMatchInfo{
		MatchID:       raw.Metadata.MatchID,
		GameCreation:  raw.Info.GameCreation,
		GameDuration:  raw.Info.GameDuration,
		GameMode:      raw.Info.GameMode,
		Participants:  make([]MatchParticipant, len(raw.Info.Participants)),
	}

	for i, p := range raw.Info.Participants {
		result.Participants[i] = MatchParticipant{
			RiotIDGameName: p.RiotIDGameName,
			RiotIDTagline:  p.RiotIDTagline,
			ChampionName:   p.ChampionName,
			ChampionID:     p.ChampionID,
			TeamID:         p.TeamID,
			Win:            p.Win,
			Kills:          p.Kills,
			Deaths:         p.Deaths,
			Assists:        p.Assists,
			Role:           p.Role,
			Lane:           p.Lane,
		}
	}

	return result, nil
}

// LeagueEntry represents a ranked league entry
type LeagueEntry struct {
	QueueType    string `json:"queueType"`
	Tier         string `json:"tier"`
	Rank         string `json:"rank"`
	LeaguePoints int    `json:"leaguePoints"`
	Wins         int    `json:"wins"`
	Losses       int    `json:"losses"`
	HotStreak    bool   `json:"hotStreak"`
	PlayerOrTeamName string `json:"playerOrTeamName"`
	PUUID        string `json:"puuid,omitempty"`
}

// GetLeagueEntriesByPUUID fetches all ranked entries for a player by PUUID
// Uses platform URL, region maps to platform
func (c *RiotClient) GetLeagueEntriesByPUUID(region, puuid string) ([]LeagueEntry, error) {
	baseURL := c.getPlatformBaseURL(region)
	path := fmt.Sprintf("/lol/league/v4/entries/by-puuid/%s", puuid)
	data, err := c.doRequest(baseURL, path)
	if err != nil {
		return nil, err
	}
	var entries []LeagueEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse league entries failed: %w", err)
	}
	return entries, nil
}

// TopPlayer represents a player in challenger/grandmaster/master leaderboard
type TopPlayer struct {
	PUUID         string `json:"puuid"`
	SummonerName  string `json:"summonerName"`
	LeaguePoints  int    `json:"leaguePoints"`
	Rank          string `json:"rank"`
	Tier          string `json:"tier"`
	Wins          int    `json:"wins"`
	Losses        int    `json:"losses"`
	HotStreak     bool   `json:"hotStreak"`
}

// GetTopPlayers fetches challenger/grandmaster/master leaderboard
func (c *RiotClient) GetTopPlayers(region, queue, tier string) ([]TopPlayer, error) {
	baseURL := c.getPlatformBaseURL(region)
	path := fmt.Sprintf("/lol/league/v4/%s/by-queue/%s", tier, queue)
	data, err := c.doRequest(baseURL, path)
	if err != nil {
		return nil, err
	}
	// Challenger/Grandmaster/Master response is a league object with entries array
	var league struct {
		Entries []struct {
			PUUID         string `json:"puuid"`
			SummonerName  string `json:"summonerName"`
			LeaguePoints  int    `json:"leaguePoints"`
			Rank          string `json:"rank"`
			Tier          string `json:"tier"`
			Wins          int    `json:"wins"`
			Losses        int    `json:"losses"`
			HotStreak     bool   `json:"hotStreak"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(data, &league); err != nil {
		return nil, fmt.Errorf("parse top players failed: %w", err)
	}
	result := make([]TopPlayer, len(league.Entries))
	for i, e := range league.Entries {
		result[i] = TopPlayer{
			PUUID:        e.PUUID,
			SummonerName: e.SummonerName,
			LeaguePoints: e.LeaguePoints,
			Rank:         e.Rank,
			Tier:         e.Tier,
			Wins:         e.Wins,
			Losses:       e.Losses,
			HotStreak:    e.HotStreak,
		}
	}
	return result, nil
}

// ServerStatus represents a platform's status
type ServerStatus struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Locales   []string `json:"locales"`
	Maintenances []struct {
		ID          any `json:"id"`
		MaintenanceType string `json:"maintenanceType"`
		Status      string `json:"status"`
		Titles      []struct {
			Locale string `json:"locale"`
			Content string `json:"content"`
		} `json:"titles"`
	} `json:"maintenances"`
	Incidents []struct {
		ID       any `json:"id"`
		Status   string `json:"status"`
		IncidentType string `json:"incidentType"`
		Title    string `json:"title"`
	} `json:"incidents"`
}

// GetServerStatus fetches server status for a region
func (c *RiotClient) GetServerStatus(region string) (*ServerStatus, error) {
	baseURL := c.getPlatformBaseURL(region)
	data, err := c.doRequest(baseURL, "/lol/status/v4/platform-data")
	if err != nil {
		return nil, err
	}
	var status ServerStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("parse server status failed: %w", err)
	}
	return &status, nil
}

// PlayerChallenge represents a player's challenge stats
type PlayerChallenge struct {
	ChallengeID      int64   `json:"challengeId"`
	Level            string  `json:"level"`   // IRON, BRONZE, ..., CHALLENGER
	CurrentValue     int64   `json:"currentValue"`
	Percentile       float64 `json:"percentile"`
}

// GetPlayerChallenges fetches challenge stats for a player
func (c *RiotClient) GetPlayerChallenges(region, puuid string) ([]PlayerChallenge, error) {
	baseURL := c.getPlatformBaseURL(region)
	path := fmt.Sprintf("/lol/challenges/v1/player-data/%s", puuid)
	data, err := c.doRequest(baseURL, path)
	if err != nil {
		return nil, err
	}
	// Response is an object with challenge info
	var response struct {
		Challenges []PlayerChallenge `json:"challenges"`
		// Also has totalPoints, rank etc.
		TotalPoints struct {
			Level     string `json:"level"`
			CurrentValue int64 `json:"currentValue"`
			Percentile float64 `json:"percentile"`
		} `json:"totalPoints"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse challenges failed: %w", err)
	}
	return response.Challenges, nil
}

// ClashPlayer represents a player's clash registration
type ClashPlayer struct {
	PUUID         string `json:"puuid"`
	SummonerName  string `json:"summonerName"`
	TeamID        string `json:"teamId"`
	Position      string `json:"position"` // TOP, JGL, MID, ADC, SUP
	Role          string `json:"role"`
}

// GetClashPlayers fetches clash registrations for a player
func (c *RiotClient) GetClashPlayers(region, puuid string) ([]ClashPlayer, error) {
	baseURL := c.getPlatformBaseURL(region)
	path := fmt.Sprintf("/lol/clash/v1/players/by-puuid/%s", puuid)
	data, err := c.doRequest(baseURL, path)
	if err != nil {
		return nil, err
	}
	var players []ClashPlayer
	if err := json.Unmarshal(data, &players); err != nil {
		return nil, fmt.Errorf("parse clash players failed: %w", err)
	}
	return players, nil
}

// ActiveGame represents a live/spectator game
type ActiveGame struct {
	GameID            int64  `json:"gameId"`
	GameType          string `json:"gameType"`
	GameStartTime     int64  `json:"gameStartTime"`
	MapID             int64  `json:"mapId"`
	GameLength        int64  `json:"gameLength"`
	GameMode          string `json:"gameMode"`
	GameQueueConfigID int64  `json:"gameQueueConfigId"`
	Participants      []ActiveGameParticipant `json:"participants"`
	BannedChampions   []BannedChampion `json:"bannedChampions"`
}

type ActiveGameParticipant struct {
	PUUID        string `json:"puuid"`
	ChampionID   int64  `json:"championId"`
	TeamID       int64  `json:"teamId"`
	SummonerName string `json:"summonerName"`
	RiotID       string `json:"riotId"`
	Spell1ID     int64  `json:"spell1Id"`
	Spell2ID     int64  `json:"spell2Id"`
}

type BannedChampion struct {
	ChampionID int64 `json:"championId"`
	TeamID     int64 `json:"teamId"`
	PickTurn   int   `json:"pickTurn"`
}

// GetActiveGame fetches the active (live) game for a PUUID
func (c *RiotClient) GetActiveGame(region, puuid string) (*ActiveGame, error) {
	baseURL := c.getPlatformBaseURL(region)
	path := fmt.Sprintf("/lol/spectator/v5/active-games/by-summoner/%s", puuid)
	data, err := c.doRequest(baseURL, path)
	if err != nil {
		return nil, err
	}
	var game ActiveGame
	if err := json.Unmarshal(data, &game); err != nil {
		return nil, fmt.Errorf("parse active game failed: %w", err)
	}
	return &game, nil
}

// FeaturedGames represents the featured games response
type FeaturedGames struct {
	GameList []ActiveGame `json:"gameList"`
}

// GetFeaturedGames fetches featured games for a region
func (c *RiotClient) GetFeaturedGames(region string) (*FeaturedGames, error) {
	baseURL := c.getPlatformBaseURL(region)
	data, err := c.doRequest(baseURL, "/lol/spectator/v5/featured-games")
	if err != nil {
		return nil, err
	}
	var featured FeaturedGames
	if err := json.Unmarshal(data, &featured); err != nil {
		return nil, fmt.Errorf("parse featured games failed: %w", err)
	}
	return &featured, nil
}

// ChampionRotation represents champion rotation info
type ChampionRotation struct {
	FreeChampionIDs              []int `json:"freeChampionIds"`
	FreeChampionIDsForNewPlayers []int `json:"freeChampionIdsForNewPlayers"`
	MaxNewPlayerLevel            int   `json:"maxNewPlayerLevel"`
}

// GetChampionRotation fetches current free champion rotation
func (c *RiotClient) GetChampionRotation(region string) (*ChampionRotation, error) {
	baseURL := c.getPlatformBaseURL(region)
	data, err := c.doRequest(baseURL, "/lol/platform/v3/champion-rotations")
	if err != nil {
		return nil, err
	}
	var rotation ChampionRotation
	if err := json.Unmarshal(data, &rotation); err != nil {
		return nil, fmt.Errorf("parse champion rotation failed: %w", err)
	}
	return &rotation, nil
}

// MatchTimeline represents a match timeline
type MatchTimeline struct {
	Metadata struct {
		MatchID      string   `json:"matchId"`
		Participants []string `json:"participants"`
	} `json:"metadata"`
	Info struct {
		FrameInterval int `json:"frameInterval"`
		Frames        []TimelineFrame `json:"frames"`
	} `json:"info"`
}

type TimelineFrame struct {
	Timestamp        int `json:"timestamp"`
	ParticipantFrames map[string]ParticipantFrame `json:"participantFrames"`
	Events            []TimelineEvent `json:"events"`
}

type ParticipantFrame struct {
	ParticipantID       int `json:"participantId"`
	TotalGold           int `json:"totalGold"`
	CurrentGold         int `json:"currentGold"`
	Level               int `json:"level"`
	XP                  int `json:"xp"`
	MinionsKilled       int `json:"minionsKilled"`
	JungleMinionsKilled int `json:"jungleMinionsKilled"`
	Position            struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"position"`
}

type TimelineEvent struct {
	Timestamp    int    `json:"timestamp"`
	Type         string `json:"type"`
	KillerID     int    `json:"killerId,omitempty"`
	VictimID     int    `json:"victimId,omitempty"`
	AssistingIDs []int  `json:"assistingParticipantIds,omitempty"`
	ItemID       int    `json:"itemId,omitempty"`
	MonsterType  string `json:"monsterType,omitempty"`
}

// GetMatchTimeline fetches timeline data for a match
func (c *RiotClient) GetMatchTimeline(region, matchID string) (*MatchTimeline, error) {
	baseURL := c.getRegionalBaseURL(region)
	path := fmt.Sprintf("/lol/match/v5/matches/%s/timeline", matchID)
	data, err := c.doRequest(baseURL, path)
	if err != nil {
		return nil, err
	}
	var timeline MatchTimeline
	if err := json.Unmarshal(data, &timeline); err != nil {
		return nil, fmt.Errorf("parse match timeline failed: %w", err)
	}
	return &timeline, nil
}
