//go:build !windows

package monitor

import (
	"context"
	"runtime/metrics"
	"syscall"
	"time"

	"github.com/sassoftware/gopher-hole/util/math"
	"github.com/sassoftware/gopher-hole/util/safebuffer"
)

const (
	procStatsHistoryDuration     = 60 * time.Second
	procStatsCollectionFrequency = 5 * time.Second

	// Number of decimal places to round availability metrics to
	roundingPrecision = 4
)

var (
	cpuMillicoreHistory = safebuffer.New(int(procStatsHistoryDuration.Seconds() / procStatsCollectionFrequency.Seconds()))
	memHistory          = safebuffer.New(int(procStatsHistoryDuration.Seconds() / procStatsCollectionFrequency.Seconds()))
	maxMemory           float64
	cpuMillicoreLimit   int
)

type ProcessStats struct {
	CPUAvailability    float64 // CPU availability as a ratio (0.0 to 1.0)
	MemoryAvailability float64 // Memory availability as a ratio (0.0 to 1.0)
}

type cpuProcStat struct {
	uTime      time.Duration
	sTime      time.Duration
	gatherTime time.Time
}

func GetMemoryAvailability() float64 {
	memData := memHistory.GetData()
	if len(memData) == 0 {
		return -1.0 // Unknown availability if we have no data
	}

	var memTotal uint64
	for _, mem := range memData {
		memTotal += mem.(uint64)
	}
	memAvg := float64(memTotal) / float64(len(memData))
	if maxMemory > 0 {
		memoryUsageRate := memAvg / maxMemory
		memoryAvail := 1 - memoryUsageRate
		if memoryAvail < 0 {
			memoryAvail = 0 // Ensure availability doesn't go negative
		}
		return math.RoundFloat(memoryAvail, roundingPrecision) // Round to 4 decimal places for consistency
	}
	// If we don't know max memory, we can't calculate availability, so return -1 to indicate unknown
	return -1.0
}

func GetCPUAvailability() float64 {
	cpuData := cpuMillicoreHistory.GetData()
	if len(cpuData) == 0 {
		return -1.0 // Unknown availability if we have no data
	}

	if cpuMillicoreLimit <= 0 {
		return -1.0 // Unknown availability if we don't know the limit
	}

	var cpuMillicoreTotal int
	for _, usage := range cpuData {
		cpuMillicoreTotal += usage.(int)
	}
	cpuMillicoreAvg := float64(cpuMillicoreTotal) / float64(len(cpuData))
	millicoreUsage := cpuMillicoreAvg / float64(cpuMillicoreLimit)
	logger.Trace().Msgf("cpuMillicoreHistory: %v", cpuMillicoreHistory)
	logger.Trace().Msgf("cpuMillicoreAvg: %v, cpuMillicoreLimit: %v, percentageOfAllocatedMillicores: %v", cpuMillicoreAvg, cpuMillicoreLimit, millicoreUsage*100)

	// Calculate CPU availability (1.0 means fully available, 0.0 means fully utilized)
	// Assume 100% per CPU core as maximum, so availability = (100 - usage_percentage) / 100
	cpuAvail := 1.0 - millicoreUsage
	if cpuAvail < 0 {
		cpuAvail = 0 // Ensure availability doesn't go negative
	}
	return math.RoundFloat(cpuAvail, roundingPrecision) // Round to 4 decimal places for consistency
}

func processMonitorNonWindows(ctx context.Context) {
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
		memHistory.Add(totalBytes)

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
					cpuMillicoreHistory.Add(millicoreUsage)
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
