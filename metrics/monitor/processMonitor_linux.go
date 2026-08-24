package monitor

import (
	"context"
)

func MonitorProcessStats(ctx context.Context) {
	readCgroupLimits()
	processMonitorNonWindows(ctx)
}
