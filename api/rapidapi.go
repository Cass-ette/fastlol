package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	rapidAPIHost = "lol-champion-stat.p.rapidapi.com"
	baseURL      = "https://" + rapidAPIHost
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) doRequest(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-rapidapi-host", rapidAPIHost)
	req.Header.Set("x-rapidapi-key", c.apiKey)

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

// ChampionCounter represents counter data for a champion
type ChampionCounter struct {
	Name       string   `json:"name"`
	WinRate    float64  `json:"winRate"`
	StrongAgainst []MatchupInfo `json:"strongAgainst"`
	WeakAgainst   []MatchupInfo `json:"weakAgainst"`
}

type MatchupInfo struct {
	Name    string  `json:"name"`
	WinRate float64 `json:"winRate"`
}

// ChampionRanking represents a champion in the tier/ranking list
type ChampionRanking struct {
	Name    string  `json:"name"`
	Tier    string  `json:"tier"`
	Role    string  `json:"role"`
	WinRate float64 `json:"winRate"`
	PickRate float64 `json:"pickRate"`
	BanRate  float64 `json:"banRate"`
	KDA     float64 `json:"kda"`
}

// GetRanking fetches the champion tier/ranking list
func (c *Client) GetRanking() ([]ChampionRanking, error) {
	data, err := c.doRequest("/champ_stat/ranking")
	if err != nil {
		return nil, err
	}

	var rankings []ChampionRanking
	if err := json.Unmarshal(data, &rankings); err != nil {
		// Try to parse as a wrapper object
		var wrapper map[string]json.RawMessage
		if err2 := json.Unmarshal(data, &wrapper); err2 != nil {
			return nil, fmt.Errorf("parse response failed: %w (raw: %s)", err, truncate(string(data), 200))
		}
		// Try common wrapper keys
		for _, key := range []string{"data", "champions", "ranking", "results"} {
			if raw, ok := wrapper[key]; ok {
				if err3 := json.Unmarshal(raw, &rankings); err3 == nil {
					return rankings, nil
				}
			}
		}
		return nil, fmt.Errorf("unexpected response format: %s", truncate(string(data), 200))
	}
	return rankings, nil
}

// GetChampionStat fetches stats for a specific champion
func (c *Client) GetChampionStat(championName string) (json.RawMessage, error) {
	name := strings.ToLower(strings.ReplaceAll(championName, " ", ""))
	data, err := c.doRequest("/champ_stat/" + name)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
