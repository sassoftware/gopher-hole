//go:build darwin

package metrics

import (
	"context"
	"runtime"
)

func MonitorProcessStats(ctx context.Context) {
	cpuMillicoreLimit = runtime.NumCPU() * 1000
	ProcessMonitorNonWindows(ctx)
}
