//go:build darwin

package monitor

import (
	"context"
	"runtime"
)

func MonitorProcessStats(ctx context.Context) {
	cpuMillicoreLimit = runtime.NumCPU() * 1000
	processMonitorNonWindows(ctx)
}
