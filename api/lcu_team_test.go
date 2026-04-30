package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func newTestLCUTeamClient(t *testing.T, handler http.Handler) (*LCUClient, string) {
	t.Helper()

	const token = "team-test-token"
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("parse test server port: %v", err)
	}

	info := LCUConnectionInfo{
		Protocol: parsed.Scheme,
		Port:     port,
		Username: "riot",
		Password: token,
	}
	return NewLCUClient(info), token
}

func TestLCUClientGetLobbyTeam(t *testing.T) {
	client, token := newTestLCUTeamClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/lol-lobby/v2/lobby" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != "riot" || password != "team-test-token" {
			t.Fatalf("basic auth mismatch")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"localMember": {
				"summonerName": "LocalPlayer",
				"gameName": "LocalGame",
				"tagLine": "LOCAL",
				"puuid": "local-puuid-123",
				"summonerId": 111
			},
			"members": [
				{
					"summonerName": "LocalPlayer",
					"gameName": "LocalGame",
					"tagLine": "LOCAL",
					"puuid": "local-puuid-123",
					"summonerId": 111
				},
				{
					"summonerName": "AllySummoner",
					"gameName": "AllyGame",
					"tagLine": "53956",
					"puuid": "ally-puuid-456",
					"summonerId": 222
				}
			]
		}`))
	}))

	members, err := client.GetLobbyTeam()
	if err != nil {
		if strings.Contains(err.Error(), token) {
			t.Fatalf("GetLobbyTeam error leaked token")
		}
		t.Fatalf("GetLobbyTeam failed: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	if members[0].SummonerName != "LocalPlayer" {
		t.Fatalf("expected first member to be LocalPlayer, got %q", members[0].SummonerName)
	}
	if !members[0].LocalPlayer {
		t.Fatalf("expected first member to be marked local")
	}
	if got := members[1].DisplayName(); got != "AllyGame#53956" {
		t.Fatalf("expected ally display name AllyGame#53956, got %q", got)
	}
	if members[1].PUUID != "ally-puuid-456" {
		t.Fatalf("expected ally PUUID to be parsed, got %q", members[1].PUUID)
	}
	if members[1].Source != "lobby" {
		t.Fatalf("expected ally source lobby, got %q", members[1].Source)
	}
}

func TestLCUClientGetLobbyTeamAddsLocalMemberWhenMembersOmitIt(t *testing.T) {
	client, _ := newTestLCUTeamClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "riot" || password != "team-test-token" {
			t.Fatalf("basic auth mismatch")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"localMember": {
				"summonerName": "LocalPlayer",
				"gameName": "LocalGame",
				"tagLine": "LOCAL",
				"puuid": "local-puuid-123",
				"summonerId": 111
			},
			"members": [
				{
					"summonerName": "AllySummoner",
					"gameName": "AllyGame",
					"tagLine": "53956",
					"puuid": "ally-puuid-456",
					"summonerId": 222
				}
			]
		}`))
	}))

	members, err := client.GetLobbyTeam()
	if err != nil {
		t.Fatalf("GetLobbyTeam failed: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	if members[0].SummonerName != "LocalPlayer" || !members[0].LocalPlayer {
		t.Fatalf("expected local member first and marked local, got %+v", members[0])
	}
	if members[1].SummonerName != "AllySummoner" || members[1].LocalPlayer {
		t.Fatalf("expected ally second and non-local, got %+v", members[1])
	}
}

func TestLCUClientGetLobbyTeamHTTPErrorDoesNotLeakToken(t *testing.T) {
	client, token := newTestLCUTeamClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))

	_, err := client.GetLobbyTeam()
	if err == nil {
		t.Fatalf("expected HTTP error")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("HTTP error leaked token")
	}
}

func TestLCUClientGetChampSelectTeam(t *testing.T) {
	client, token := newTestLCUTeamClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/lol-champ-select/v1/session" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"localPlayerCellId": 1,
			"myTeam": [
				{
					"cellId": 1,
					"championId": 157,
					"assignedPosition": "middle",
					"puuid": "my-puuid",
					"summonerId": 100,
					"gameName": "Me",
					"tagLine": "CN1"
				},
				{
					"cellId": 2,
					"championId": 64,
					"assignedPosition": "jungle",
					"puuid": "ally-puuid",
					"summonerId": 200,
					"gameName": "Ally",
					"tagLine": "53956"
				}
			]
		}`))
	}))

	members, err := client.GetChampSelectTeam()
	if err != nil {
		if strings.Contains(err.Error(), token) {
			t.Fatalf("GetChampSelectTeam error leaked token")
		}
		t.Fatalf("GetChampSelectTeam failed: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	if !members[0].LocalPlayer {
		t.Fatalf("expected first member to be marked local")
	}
	if members[1].ChampionID != 64 {
		t.Fatalf("expected second member champion ID 64, got %d", members[1].ChampionID)
	}
	if members[1].AssignedPosition != "jungle" {
		t.Fatalf("expected second member assigned position jungle, got %q", members[1].AssignedPosition)
	}
	if members[1].Source != "champ-select" {
		t.Fatalf("expected second member source champ-select, got %q", members[1].Source)
	}
}

func TestLCUClientGetCurrentTeamUsesChampSelectPhase(t *testing.T) {
	client, _ := newTestLCUTeamClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/lol-gameflow/v1/gameflow-phase":
			_, _ = w.Write([]byte(strconv.Quote("ChampSelect")))
		case "/lol-champ-select/v1/session":
			_, _ = w.Write([]byte(`{
				"localPlayerCellId": 1,
				"myTeam": [
					{
						"cellId": 1,
						"championId": 157,
						"assignedPosition": "middle",
						"puuid": "my-puuid",
						"summonerId": 100,
						"gameName": "Me",
						"tagLine": "CN1"
					}
				]
			}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))

	snapshot, err := client.GetCurrentTeam()
	if err != nil {
		t.Fatalf("GetCurrentTeam failed: %v", err)
	}
	if snapshot.Phase != "ChampSelect" {
		t.Fatalf("expected phase ChampSelect, got %q", snapshot.Phase)
	}
	if snapshot.Source != "champ-select" {
		t.Fatalf("expected source champ-select, got %q", snapshot.Source)
	}
	if len(snapshot.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(snapshot.Members))
	}
}

func TestLCUClientGetCurrentTeamUsesLobbyLikePhases(t *testing.T) {
	for _, phase := range []string{"Lobby", "Matchmaking", "ReadyCheck"} {
		t.Run(phase, func(t *testing.T) {
			client, _ := newTestLCUTeamClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/lol-gameflow/v1/gameflow-phase":
					_, _ = w.Write([]byte(strconv.Quote(phase)))
				case "/lol-lobby/v2/lobby":
					_, _ = w.Write([]byte(`{
						"localMember": {
							"summonerName": "LocalPlayer",
							"gameName": "LocalGame",
							"tagLine": "LOCAL",
							"puuid": "local-puuid-123",
							"summonerId": 111
						},
						"members": [
							{
								"summonerName": "LocalPlayer",
								"gameName": "LocalGame",
								"tagLine": "LOCAL",
								"puuid": "local-puuid-123",
								"summonerId": 111
							}
						]
					}`))
				default:
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
			}))

			snapshot, err := client.GetCurrentTeam()
			if err != nil {
				t.Fatalf("GetCurrentTeam failed: %v", err)
			}
			if snapshot.Phase != phase {
				t.Fatalf("expected phase %s, got %q", phase, snapshot.Phase)
			}
			if snapshot.Source != "lobby" {
				t.Fatalf("expected source lobby, got %q", snapshot.Source)
			}
			if len(snapshot.Members) != 1 {
				t.Fatalf("expected 1 member, got %d", len(snapshot.Members))
			}
		})
	}
}

func TestLCUClientGetCurrentTeamUnavailableDoesNotLeakToken(t *testing.T) {
	client, token := newTestLCUTeamClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lol-gameflow/v1/gameflow-phase" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(strconv.Quote("None")))
	}))

	_, err := client.GetCurrentTeam()
	if err == nil {
		t.Fatalf("expected unavailable phase error")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("GetCurrentTeam error leaked token")
	}
	if !strings.Contains(err.Error(), "None") {
		t.Fatalf("expected error to mention None phase, got %v", err)
	}
}
