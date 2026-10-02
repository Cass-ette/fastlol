package api

import (
	"fmt"
	"strings"
)

// LCUTeamMember is a normalized read-only view of a League Client team member.
type LCUTeamMember struct {
	SummonerName     string
	GameName         string
	TagLine          string
	PUUID            string
	SummonerID       int64
	ChampionID       int
	ChampionName     string
	AssignedPosition string
	CellID           int
	LocalPlayer      bool
	Source           string
}

// DisplayName returns the best token-free display name available for a team member.
func (m LCUTeamMember) DisplayName() string {
	if m.GameName != "" && m.TagLine != "" {
		return m.GameName + "#" + m.TagLine
	}
	if m.GameName != "" {
		return m.GameName
	}
	if m.SummonerName != "" {
		return m.SummonerName
	}
	if id := shortTokenFreeID(m.PUUID); id != "" {
		return id
	}
	return "Unknown"
}

// LCUTeamSnapshot captures a read-only team view from an LCU source.
type LCUTeamSnapshot struct {
	Phase   string
	Source  string
	Members []LCUTeamMember
}

type lcuLobbyResponse struct {
	LocalMember lcuLobbyMember   `json:"localMember"`
	Members     []lcuLobbyMember `json:"members"`
}

type lcuLobbyMember struct {
	SummonerName string `json:"summonerName"`
	GameName     string `json:"gameName"`
	TagLine      string `json:"tagLine"`
	PUUID        string `json:"puuid"`
	SummonerID   int64  `json:"summonerId"`
}

type lcuChampSelectSession struct {
	LocalPlayerCellID int                       `json:"localPlayerCellId"`
	MyTeam            []lcuChampSelectTeamEntry `json:"myTeam"`
}

type lcuChampSelectTeamEntry struct {
	CellID           int    `json:"cellId"`
	ChampionID       int    `json:"championId"`
	ChampionName     string `json:"championName"`
	AssignedPosition string `json:"assignedPosition"`
	PUUID            string `json:"puuid"`
	SummonerID       int64  `json:"summonerId"`
	SummonerName     string `json:"summonerName"`
	GameName         string `json:"gameName"`
	TagLine          string `json:"tagLine"`
}

// GetGameflowPhase returns the current League Client gameflow phase.
func (c *LCUClient) GetGameflowPhase() (string, error) {
	var phase string
	if err := c.GetJSON("/lol-gameflow/v1/gameflow-phase", &phase); err != nil {
		return "", fmt.Errorf("get LCU gameflow phase: %w", err)
	}
	phase = strings.TrimSpace(phase)
	if phase == "" {
		return "Unknown", nil
	}
	return phase, nil
}

// GetLobbyTeam returns the current lobby members from the read-only LCU lobby endpoint.
func (c *LCUClient) GetLobbyTeam() ([]LCUTeamMember, error) {
	var lobby lcuLobbyResponse
	if err := c.GetJSON("/lol-lobby/v2/lobby", &lobby); err != nil {
		return nil, fmt.Errorf("get LCU lobby team: %w", err)
	}

	localKey := lobby.LocalMember.identityKey()
	seen := make(map[string]struct{})
	members := make([]LCUTeamMember, 0, len(lobby.Members)+1)

	addMember := func(member lcuLobbyMember) {
		key := member.identityKey()
		if key != "" {
			if _, ok := seen[key]; ok {
				return
			}
			seen[key] = struct{}{}
		}

		teamMember := member.toTeamMember()
		teamMember.LocalPlayer = key != "" && key == localKey
		members = append(members, teamMember)
	}

	if localKey != "" {
		addMember(lobby.LocalMember)
	}
	for _, member := range lobby.Members {
		addMember(member)
	}

	return members, nil
}

func (m lcuLobbyMember) toTeamMember() LCUTeamMember {
	return LCUTeamMember{
		SummonerName: m.SummonerName,
		GameName:     m.GameName,
		TagLine:      m.TagLine,
		PUUID:        m.PUUID,
		SummonerID:   m.SummonerID,
		Source:       "lobby",
	}
}

// GetChampSelectTeam returns the current champ-select team from the read-only LCU session endpoint.
func (c *LCUClient) GetChampSelectTeam() ([]LCUTeamMember, error) {
	var session lcuChampSelectSession
	if err := c.GetJSON("/lol-champ-select/v1/session", &session); err != nil {
		return nil, fmt.Errorf("get LCU champ-select team: %w", err)
	}

	members := make([]LCUTeamMember, 0, len(session.MyTeam))
	for _, entry := range session.MyTeam {
		members = append(members, LCUTeamMember{
			SummonerName:     entry.SummonerName,
			GameName:         entry.GameName,
			TagLine:          entry.TagLine,
			PUUID:            entry.PUUID,
			SummonerID:       entry.SummonerID,
			ChampionID:       entry.ChampionID,
			ChampionName:     entry.ChampionName,
			AssignedPosition: entry.AssignedPosition,
			CellID:           entry.CellID,
			LocalPlayer:      entry.CellID == session.LocalPlayerCellID,
			Source:           "champ-select",
		})
	}
	return members, nil
}

// GetCurrentTeam returns the current team from the appropriate read-only LCU endpoint for the phase.
func (c *LCUClient) GetCurrentTeam() (LCUTeamSnapshot, error) {
	phase, err := c.GetGameflowPhase()
	if err != nil {
		return LCUTeamSnapshot{}, err
	}

	if strings.EqualFold(strings.TrimSpace(phase), "ChampSelect") {
		members, err := c.GetChampSelectTeam()
		if err != nil {
			return LCUTeamSnapshot{}, err
		}
		return LCUTeamSnapshot{Phase: phase, Source: "champ-select", Members: members}, nil
	}

	if isLCULobbyLikePhase(phase) {
		members, err := c.GetLobbyTeam()
		if err != nil {
			return LCUTeamSnapshot{}, err
		}
		return LCUTeamSnapshot{Phase: phase, Source: "lobby", Members: members}, nil
	}

	return LCUTeamSnapshot{}, fmt.Errorf("LCU team unavailable in gameflow phase %s; enter a lobby or champ select", phase)
}

func isLCULobbyLikePhase(phase string) bool {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "lobby", "matchmaking", "readycheck":
		return true
	default:
		return false
	}
}

func (m lcuLobbyMember) identityKey() string {
	if m.PUUID != "" {
		return "puuid:" + m.PUUID
	}
	if m.SummonerID != 0 {
		return fmt.Sprintf("summoner:%d", m.SummonerID)
	}
	if m.GameName != "" || m.TagLine != "" {
		return "riotid:" + strings.ToLower(m.GameName+"#"+m.TagLine)
	}
	if m.SummonerName != "" {
		return "name:" + strings.ToLower(m.SummonerName)
	}
	return ""
}

func shortTokenFreeID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}
