package metrics

import (
	"context"
)

func MonitorProcessStats(ctx context.Context) {
	readCgroupLimits()
	ProcessMonitorNonWindows(ctx)
}
