package metrics

import (
	"context"
	"time"

	"github.com/sassoftware/gopher-hole/math"
	"github.com/sassoftware/gopher-hole/safebuffer"
)

const (
	ServiceCPUAvailableMetricName    = "service_cpu_available"
	ServiceMemoryAvailableMetricName = "service_memory_available"

	// constants for process stats monitoring
	procStatsHistoryDuration     = 60 * time.Second
	procStatsCollectionFrequency = 5 * time.Second

	// Define thresholds for unhealthy usage
	cpuAvailabilityThreshold    = 0.1 // 10% CPU availability
	memoryAvailabilityThreshold = 0.1 // 10% Memory availability

	// Number of decimal places to round availability metrics to
	roundingPrecision = 4
)

var (
	TimeBetweenCollections = time.Minute
	cpuMillicoreHistory    = safebuffer.NewSafeBuffer(int(procStatsHistoryDuration.Seconds() / procStatsCollectionFrequency.Seconds()))
	memHistory             = safebuffer.NewSafeBuffer(int(procStatsHistoryDuration.Seconds() / procStatsCollectionFrequency.Seconds()))
	maxMemory              float64
	cpuMillicoreLimit      int
)

type ProcessStats struct {
	CPUAvailability    float64 // CPU availability as a ratio (0.0 to 1.0)
	MemoryAvailability float64 // Memory availability as a ratio (0.0 to 1.0)
}

func setMaxMemory(mem float64) {
	maxMemory = mem
}

func setCPUMillicoreLimit(millicores int) {
	cpuMillicoreLimit = millicores
}

func addMemoryHistory(mem uint64) { //nolint:unused
	memHistory.Add(mem)
}

func addCPUHistory(cpu int) { //nolint:unused
	cpuMillicoreHistory.Add(cpu)
}

func CollectBaseServiceMetrics(ctx context.Context) error {
	// Get the metrics manager from context
	mgr, err := GetManagerFromContext(ctx)
	if err != nil {
		return err
	}

	// Create and register CPU and Memory availability metrics
	cpuAvailMetric := NewMetric(ServiceCPUAvailableMetricName, []string{SystemLabel})
	err = mgr.Register(cpuAvailMetric)
	if err != nil {
		return err
	}
	memAvailMetric := NewMetric(ServiceMemoryAvailableMetricName, []string{SystemLabel})
	err = mgr.Register(memAvailMetric)
	if err != nil {
		return err
	}

	// Start a goroutine to continuously collect CPU and memory usage samples
	go MonitorProcessStats(ctx)

	// Start a goroutine to periodically collect and store CPU and Memory availability
	go periodicCollect(ctx, cpuAvailMetric, memAvailMetric)

	return nil
}

func periodicCollect(ctx context.Context, cpuAvailMetric, memAvailMetric *Metric) {
	ticker := time.NewTicker(TimeBetweenCollections)
	defer ticker.Stop()
	<-ticker.C // Initial wait before first collection
	for {
		procStats := getProcessStats()
		// Measure CPU availability
		if procStats.CPUAvailability == -1.0 {
			logger.Debug().Msg("Failed to measure service CPU availability")
		} else {
			cpuAvailMetric.InsertRecord(float64(procStats.CPUAvailability))
			logger.Debug().Float64(ServiceCPUAvailableMetricName, procStats.CPUAvailability).Msg("Service CPU availability metric collected")
		}

		// Measure Memory availability
		if procStats.MemoryAvailability == -1.0 {
			logger.Debug().Msg("Failed to measure service memory availability")
		} else {
			memAvailMetric.InsertRecord(float64(procStats.MemoryAvailability))
			logger.Debug().Float64(ServiceMemoryAvailableMetricName, procStats.MemoryAvailability).Msg("Service memory availability metric collected")
		}

		// Wait for a constant amount of time between collections
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func memoryAvailability() float64 {
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

func cpuAvailability() float64 {
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

func getProcessStats() *ProcessStats {
	ps := &ProcessStats{}

	ps.CPUAvailability = cpuAvailability()
	ps.MemoryAvailability = memoryAvailability()

	return ps
}

func (ps *ProcessStats) IsUnhealthyUsage(ctx context.Context) bool {
	if ps.CPUAvailability == -1.0 {
		logger.Trace().Msg("Unable to determine CPU availability")
	}

	if ps.MemoryAvailability == -1.0 {
		logger.Trace().Msg("Unable to determine Memory availability")
	}

	if ps.CPUAvailability >= 0 && ps.CPUAvailability < cpuAvailabilityThreshold {
		return true
	}

	if ps.MemoryAvailability >= 0 && ps.MemoryAvailability < memoryAvailabilityThreshold {
		return true
	}

	return false
}

type cpuProcStat struct {
	uTime      time.Duration
	sTime      time.Duration
	gatherTime time.Time
}
