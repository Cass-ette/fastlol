package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// ServerClient connects to a custom fastlol API server
type ServerClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewServerClient creates a client for custom fastlol API server
// baseURL should be like "http://156.225.20.57:8080"
func NewServerClient(baseURL string) *ServerClient {
	return &ServerClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *ServerClient) doRequest(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

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

// GetCounters fetches counter data from custom server
func (c *ServerClient) GetCounters(champion, role string) (*CounterData, error) {
	params := url.Values{}
	if role != "" {
		params.Set("role", role)
	}

	path := "/api/counter/" + champion
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	data, err := c.doRequest(path)
	if err != nil {
		return nil, err
	}

	var result CounterData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	return &result, nil
}

// GetMatchup fetches matchup data from custom server
func (c *ServerClient) GetMatchup(champion, enemy, role string) (*MatchupResult, error) {
	params := url.Values{}
	if role != "" {
		params.Set("role", role)
	}

	path := "/api/matchup/" + champion + "/" + enemy
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	data, err := c.doRequest(path)
	if err != nil {
		return nil, err
	}

	var result MatchupResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	return &result, nil
}
