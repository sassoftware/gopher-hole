//go:build !windows

package metrics

import (
	"context"
	"runtime/metrics"
	"syscall"
	"time"
)

func ProcessMonitorNonWindows(ctx context.Context) {
	ticker := time.NewTicker(procStatsCollectionFrequency)
	defer ticker.Stop()
	lastProcStat := &cpuProcStat{}
	for {
		ps := &cpuProcStat{}
		ps.gatherTime = time.Now()

		// Determine memory usage
		sample := []metrics.Sample{{Name: "/memory/classes/total:bytes"}}
		metrics.Read(sample)
		totalBytes := sample[0].Value.Uint64()
		addMemoryHistory(totalBytes)

		rUsage := &syscall.Rusage{}
		rUsageErr := syscall.Getrusage(syscall.RUSAGE_SELF, rUsage)
		if rUsageErr == nil {
			ps.uTime = time.Duration(rUsage.Utime.Nano())
			ps.sTime = time.Duration(rUsage.Stime.Nano())
			// Only calculate CPU usage if we have a previous stat to compare against
			if !lastProcStat.gatherTime.IsZero() {
				currTotalTime := ps.uTime + ps.sTime
				oldTotalTime := lastProcStat.uTime + lastProcStat.sTime
				totalTime := currTotalTime - oldTotalTime
				gatherTimeDiff := ps.gatherTime.Sub(lastProcStat.gatherTime)
				// Avoid division by zero if gatherTimeDiff is zero
				if gatherTimeDiff.Seconds() > 0 {
					cpuUsageAsADecimal := totalTime.Seconds() / gatherTimeDiff.Seconds() // CPU usage as a decimal (e.g., 0.25 for 25%)
					millicoreUsage := int(cpuUsageAsADecimal * 1000)
					logger.Trace().Msgf("CPU Usage: %0.2f%% (%d mCPU), Memory Sys (bytes): %d", cpuUsageAsADecimal*100, millicoreUsage, totalBytes)
					addCPUHistory(millicoreUsage)
				}
			}
			lastProcStat = ps
		}

		// Wait for the next tick or context cancellation
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}
