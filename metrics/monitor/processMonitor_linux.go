package monitor

import (
	"context"
)

func MonitorProcessStats(ctx context.Context) {
	readCgroupLimits()
	maxMemory = resolveMemoryLimit(maxMemory, getGoMemoryLimit, getNodeMemoryLimit)
	processMonitorNonWindows(ctx)
}
