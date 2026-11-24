package monitor

import (
	"context"
)

func MonitorProcessStats(ctx context.Context) {
	maxMemory = getMemoryLimit()
	processMonitorNonWindows(ctx)
}
