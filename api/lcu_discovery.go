package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const fastlolLCUTokenEnv = "FASTLOL_LCU_TOKEN"

// LCUDiscoveryOptions controls local League Client discovery.
type LCUDiscoveryOptions struct {
	Port         int
	LockfilePath string
}

// DiscoverLCU discovers local League Client Update API connection details.
func DiscoverLCU(opts LCUDiscoveryOptions) (LCUConnectionInfo, error) {
	if opts.Port > 0 {
		token := os.Getenv(fastlolLCUTokenEnv)
		if token == "" {
			return LCUConnectionInfo{}, fmt.Errorf("LCU port provided but %s is not set", fastlolLCUTokenEnv)
		}
		return LCUConnectionInfo{
			Protocol: "https",
			Port:     opts.Port,
			Username: defaultLCUUsername,
			Password: token,
		}, nil
	}

	var errs []error
	if opts.LockfilePath != "" {
		if info, err := discoverLCUFromLockfile(opts.LockfilePath); err == nil {
			return info, nil
		} else {
			errs = append(errs, fmt.Errorf("explicit lockfile: %w", err))
		}
	}

	if info, err := discoverLCUFromProcess(); err == nil {
		return info, nil
	} else {
		errs = append(errs, err)
	}

	for _, candidate := range defaultLCULockfileCandidates() {
		if info, err := discoverLCUFromLockfile(candidate); err == nil {
			return info, nil
		} else {
			errs = append(errs, err)
		}
	}

	if len(errs) == 0 {
		return LCUConnectionInfo{}, fmt.Errorf("LCU discovery failed")
	}
	return LCUConnectionInfo{}, fmt.Errorf("LCU discovery failed: %w", errors.Join(errs...))
}

func discoverLCUFromLockfile(path string) (LCUConnectionInfo, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return LCUConnectionInfo{}, err
	}
	return parseLCULockfileContent(string(content))
}

func defaultLCULockfileCandidates() []string {
	var candidates []string
	add := func(base string, elems ...string) {
		if base == "" {
			return
		}
		path := filepath.Join(append([]string{base}, elems...)...)
		if filepath.IsAbs(path) {
			candidates = append(candidates, path)
		}
	}

	add(os.Getenv("LOCALAPPDATA"), "Riot Games", "League of Legends", "lockfile")
	add(os.Getenv("LOCALAPPDATA"), "League of Legends", "lockfile")
	add(os.Getenv("PROGRAMDATA"), "Riot Games", "League of Legends", "lockfile")
	add(os.Getenv("HOME"), "Applications", "League of Legends.app", "Contents", "LoL", "lockfile")
	add("/Applications", "League of Legends.app", "Contents", "LoL", "lockfile")
	add("/Applications", "League of Legends", "lockfile")

	return candidates
}
