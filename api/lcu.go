package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const defaultLCUUsername = "riot"

// LCUConnectionInfo contains the local League Client Update API connection details.
type LCUConnectionInfo struct {
	Protocol string
	Port     int
	Username string
	Password string
	Region   string
	Tencent  bool
}

// BaseURL returns the loopback League Client API base URL.
func (i LCUConnectionInfo) BaseURL() string {
	protocol := i.Protocol
	if protocol == "" {
		protocol = "https"
	}
	return fmt.Sprintf("%s://127.0.0.1:%d", protocol, i.Port)
}

// RedactedString returns a safe representation that never includes credentials.
func (i LCUConnectionInfo) RedactedString() string {
	protocol := i.Protocol
	if protocol == "" {
		protocol = "https"
	}
	username := i.Username
	if username == "" {
		username = defaultLCUUsername
	}
	return fmt.Sprintf("LCUConnectionInfo{Protocol:%s Port:%d Username:%s Region:%s Tencent:%t AuthConfigured:%t}", protocol, i.Port, username, i.Region, i.Tencent, i.Password != "")
}

// String returns a redacted representation so accidental formatting is safer.
func (i LCUConnectionInfo) String() string {
	return i.RedactedString()
}

// GoString returns a redacted representation for Go-syntax formatting with %#v.
func (i LCUConnectionInfo) GoString() string {
	return i.RedactedString()
}

// LCUClient is a read-only client for local League Client Update API requests.
type LCUClient struct {
	info       LCUConnectionInfo
	httpClient *http.Client
}

// NewLCUClient creates a read-only local LCU API client.
func NewLCUClient(info LCUConnectionInfo) *LCUClient {
	if info.Protocol == "" {
		info.Protocol = "https"
	}
	if info.Username == "" {
		info.Username = defaultLCUUsername
	}
	return &LCUClient{
		info: info,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				// The LCU serves a self-signed certificate on loopback only; BaseURL is fixed to 127.0.0.1.
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

// GetJSON performs a read-only GET request and decodes the JSON response into out.
func (c *LCUClient) GetJSON(path string, out any) error {
	if c == nil {
		return fmt.Errorf("LCU client is nil")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	req, err := http.NewRequest(http.MethodGet, c.info.BaseURL()+path, nil)
	if err != nil {
		return fmt.Errorf("create LCU request: %w", err)
	}
	username := c.info.Username
	if username == "" {
		username = defaultLCUUsername
	}
	req.SetBasicAuth(username, c.info.Password)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("LCU request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read LCU response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("LCU returned HTTP %d", resp.StatusCode)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode LCU JSON: %w", err)
	}
	return nil
}

func parseLCULockfileContent(content string) (LCUConnectionInfo, error) {
	fields := strings.Split(strings.TrimSpace(content), ":")
	if len(fields) != 5 {
		return LCUConnectionInfo{}, fmt.Errorf("invalid LCU lockfile format")
	}
	port, err := strconv.Atoi(fields[2])
	if err != nil {
		return LCUConnectionInfo{}, fmt.Errorf("invalid LCU lockfile port")
	}
	protocol := fields[4]
	if protocol == "" {
		protocol = "https"
	}
	return LCUConnectionInfo{
		Protocol: protocol,
		Port:     port,
		Username: defaultLCUUsername,
		Password: fields[3],
	}, nil
}

func parseLCUCommandLine(commandLine string) (LCUConnectionInfo, error) {
	args := splitCommandLine(commandLine)
	var portText string
	var token string
	var tencent bool

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--app-port":
			if i+1 < len(args) {
				portText = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--app-port="):
			portText = strings.TrimPrefix(arg, "--app-port=")
		case arg == "--remoting-auth-token":
			if i+1 < len(args) {
				token = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--remoting-auth-token="):
			token = strings.TrimPrefix(arg, "--remoting-auth-token=")
		case arg == "--riotclient-tencent" || strings.HasPrefix(arg, "--riotclient-tencent="):
			tencent = true
		}
	}

	if portText == "" || token == "" {
		return LCUConnectionInfo{}, fmt.Errorf("LCU command line missing app port or auth token")
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return LCUConnectionInfo{}, fmt.Errorf("invalid LCU app port")
	}
	return LCUConnectionInfo{
		Protocol: "https",
		Port:     port,
		Username: defaultLCUUsername,
		Password: token,
		Tencent:  tencent,
	}, nil
}

func splitCommandLine(commandLine string) []string {
	var args []string
	var b strings.Builder
	inQuotes := false
	escaped := false

	flush := func() {
		if b.Len() == 0 {
			return
		}
		args = append(args, b.String())
		b.Reset()
	}

	for _, r := range commandLine {
		if escaped {
			b.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '"' {
			inQuotes = !inQuotes
			continue
		}
		if (r == ' ' || r == '\t' || r == '\n' || r == '\r') && !inQuotes {
			flush()
			continue
		}
		b.WriteRune(r)
	}
	if escaped {
		b.WriteRune('\\')
	}
	flush()
	return args
}
