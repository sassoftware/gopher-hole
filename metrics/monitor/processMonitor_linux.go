package monitor

import (
	"context"
)

func MonitorProcessStats(ctx context.Context) {
	readCgroupLimits()
	maxMemory = selectMemoryLimit(maxMemory, getGoMemoryLimit(), getNodeMemoryLimit())
	processMonitorNonWindows(ctx)
}
