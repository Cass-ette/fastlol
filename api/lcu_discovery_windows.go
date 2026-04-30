//go:build windows

package api

import (
	"fmt"
	"os/exec"
	"strings"
)

func discoverLCUFromProcess() (LCUConnectionInfo, error) {
	const command = "Get-CimInstance Win32_Process -Filter \"Name = 'LeagueClientUx.exe'\" | Select-Object -ExpandProperty CommandLine"
	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", command).Output()
	if err != nil {
		return LCUConnectionInfo{}, fmt.Errorf("read LeagueClientUx.exe command line: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		info, err := parseLCUCommandLine(line)
		if err == nil {
			return info, nil
		}
	}
	return LCUConnectionInfo{}, fmt.Errorf("LeagueClientUx.exe command line did not include LCU app port and auth token")
}
