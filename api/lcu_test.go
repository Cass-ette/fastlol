package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestParseLCULockfileUsesStandardFields(t *testing.T) {
	info, err := parseLCULockfileContent("LeagueClient:12345:29999:secret-token:https")
	if err != nil {
		t.Fatalf("parse lockfile: %v", err)
	}
	if info.Port != 29999 {
		t.Fatalf("expected LCU port 29999, got %d", info.Port)
	}
	if info.Username != "riot" {
		t.Fatalf("expected username riot, got %q", info.Username)
	}
	if info.Protocol != "https" {
		t.Fatalf("expected protocol https, got %q", info.Protocol)
	}
	if info.Password != "secret-token" {
		t.Fatalf("expected password to be parsed")
	}
}

func TestParseTencentCommandLineUsesLCUArgsOnly(t *testing.T) {
	cmd := `"C:\Riot Games\League of Legends\LeagueClientUx.exe" --riotclient-app-port=1111 --riotclient-auth-token=riot-token --riotclient-tencent --app-port=29999 --remoting-auth-token=lcu-token`

	info, err := parseLCUCommandLine(cmd)
	if err != nil {
		t.Fatalf("parse command line: %v", err)
	}
	if info.Port != 29999 {
		t.Fatalf("expected LCU app port 29999, got %d", info.Port)
	}
	if info.Password != "lcu-token" {
		t.Fatalf("expected LCU token to be parsed")
	}
	if !info.Tencent {
		t.Fatalf("expected Tencent detection")
	}
	if info.Username != "riot" {
		t.Fatalf("expected username riot, got %q", info.Username)
	}
	if info.Protocol != "https" {
		t.Fatalf("expected default protocol https, got %q", info.Protocol)
	}
}

func TestParseCommandLineDoesNotConfuseRiotClientToken(t *testing.T) {
	cmd := `"LeagueClientUx.exe" --riotclient-app-port=1111 --riotclient-auth-token=riot-token --riotclient-tencent`

	_, err := parseLCUCommandLine(cmd)
	if err == nil {
		t.Fatalf("expected error when only Riot Client args are present")
	}
	if strings.Contains(err.Error(), "riot-token") {
		t.Fatalf("error leaked Riot Client token")
	}
}

func TestLCUConnectionInfoRedactionAndString(t *testing.T) {
	info := LCUConnectionInfo{
		Protocol: "https",
		Port:     29999,
		Username: "riot",
		Password: "secret-token",
		Region:   "CN",
		Tencent:  true,
	}

	redacted := info.RedactedString()
	if strings.Contains(redacted, "secret-token") || strings.Contains(redacted, "Password") {
		t.Fatalf("redacted string leaked sensitive content: %q", redacted)
	}
	if !strings.Contains(redacted, "29999") || !strings.Contains(redacted, "CN") || !strings.Contains(redacted, "Tencent:true") {
		t.Fatalf("redacted string omitted expected non-sensitive fields: %q", redacted)
	}
	if got := info.String(); got != redacted {
		t.Fatalf("String() should return RedactedString(); got %q want %q", got, redacted)
	}
	goSyntax := fmt.Sprintf("%#v", info)
	if strings.Contains(goSyntax, "secret-token") || strings.Contains(goSyntax, "Password") {
		t.Fatalf("Go-syntax formatting leaked sensitive content: %q", goSyntax)
	}
}

func TestLCUClientGetJSONSetsBasicAuth(t *testing.T) {
	const token = "basic-auth-token"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "Basic " + base64.StdEncoding.EncodeToString([]byte("riot:"+token))
		if got := r.Header.Get("Authorization"); got != want {
			t.Fatalf("authorization header mismatch")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	info, err := connectionInfoFromTestServerURL(server.URL, token)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	client := NewLCUClient(info)

	var out struct {
		OK bool `json:"ok"`
	}
	if err := client.GetJSON("/lol-test/v1/example", &out); err != nil {
		if strings.Contains(err.Error(), token) {
			t.Fatalf("GetJSON error leaked token")
		}
		t.Fatalf("GetJSON failed: %v", err)
	}
	if !out.OK {
		t.Fatalf("expected decoded response")
	}
}

func TestLCUClientGetJSONNon2xxDoesNotLeakToken(t *testing.T) {
	const token = "non-2xx-token"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	info, err := connectionInfoFromTestServerURL(server.URL, token)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	client := NewLCUClient(info)

	var out struct{}
	err = client.GetJSON("/lol-test/v1/unauthorized", &out)
	if err == nil {
		t.Fatalf("expected non-2xx error")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("non-2xx error leaked token")
	}
}

func connectionInfoFromTestServerURL(rawURL, token string) (LCUConnectionInfo, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return LCUConnectionInfo{}, err
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		return LCUConnectionInfo{}, err
	}
	return LCUConnectionInfo{Protocol: parsed.Scheme, Port: port, Username: "riot", Password: token}, nil
}

func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}

func TestDiscoverLCUFromManualSessionValues(t *testing.T) {
	t.Setenv("FASTLOL_LCU_TOKEN", "manual-token")

	info, err := DiscoverLCU(LCUDiscoveryOptions{Port: 29999})
	if err != nil {
		t.Fatalf("discover manual LCU: %v", err)
	}
	if info.Port != 29999 {
		t.Fatalf("expected port 29999, got %d", info.Port)
	}
	if info.Password != "manual-token" {
		t.Fatalf("expected session token to be loaded")
	}
	if info.Username != "riot" {
		t.Fatalf("expected username riot, got %q", info.Username)
	}
}

func TestDiscoverLCUFromExplicitLockfilePath(t *testing.T) {
	dir := t.TempDir()
	lockfile := filepath.Join(dir, "lockfile")
	if err := writeTestFile(lockfile, "LeagueClient:12345:29999:secret-token:https"); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}

	info, err := DiscoverLCU(LCUDiscoveryOptions{LockfilePath: lockfile})
	if err != nil {
		t.Fatalf("discover explicit lockfile: %v", err)
	}
	if info.Port != 29999 {
		t.Fatalf("expected port 29999, got %d", info.Port)
	}
	if info.Password != "secret-token" {
		t.Fatalf("expected lockfile password to be parsed")
	}
}

func TestDefaultLockfileCandidatesSkipEmptyEnvRelativePaths(t *testing.T) {
	t.Setenv("LOCALAPPDATA", "")
	t.Setenv("PROGRAMDATA", "")
	t.Setenv("HOME", "")

	for _, candidate := range defaultLCULockfileCandidates() {
		if candidate == "" {
			t.Fatalf("default candidates included empty path")
		}
		if !filepath.IsAbs(candidate) {
			t.Fatalf("default candidates included relative path: %q", candidate)
		}
	}
}
