//go:build !windows

package api

import "fmt"

func discoverLCUFromProcess() (LCUConnectionInfo, error) {
	return LCUConnectionInfo{}, fmt.Errorf("LCU process discovery is only supported on Windows")
}
