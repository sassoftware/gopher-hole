package metrics

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var cgroupSyncOnce sync.Once

// CgroupVersion represents the cgroup version in use
type CgroupVersion int

var (
	cgroupV1CpuQuotaFile   = "/sys/fs/cgroup/cpu/cpu.cfs_quota_us"
	cgroupV1CpuPeriodFile  = "/sys/fs/cgroup/cpu/cpu.cfs_period_us"
	cgroupV1MemLimitFile   = "/sys/fs/cgroup/memory/memory.limit_in_bytes"
	cgroupV2CpuQuotaFile   = "/sys/fs/cgroup/cpu.max"
	cgroupV2MemLimitFile   = "/sys/fs/cgroup/memory.max"
	cgroupV2ControllerFile = "/sys/fs/cgroup/cgroup.controllers"
)

const (
	CgroupV1 CgroupVersion = iota
	CgroupV2

	noLimitIndicatorV1 = "-1"
	noLimitIndicatorV2 = "max"

	defaultCgroupCPUPeriod = 100000
)

// detectCgroupVersion determines if the system is using cgroup v1 or v2
func detectCgroupVersion() CgroupVersion {
	// Check if cgroup v2 is mounted by looking for the unified hierarchy marker
	if _, err := os.Stat(cgroupV2ControllerFile); err == nil {
		return CgroupV2
	}
	// Default to v1
	return CgroupV1
}

func readCgroupLimits() {
	cgroupSyncOnce.Do(func() {
		detectedCgroupVersion := detectCgroupVersion()
		maxMemoryVal := getMaxMemoryFromCgroup(detectedCgroupVersion)
		setMaxMemory(maxMemoryVal)
		cpuLimitVal := getCPULimitFromCgroup(detectedCgroupVersion)
		setCPUMillicoreLimit(cpuLimitVal)
	})
}

func getMaxMemoryFromCgroup(version CgroupVersion) float64 {
	var memoryLimitFile string
	var noLimitIndicator string
	if version == CgroupV2 {
		memoryLimitFile = cgroupV2MemLimitFile
		noLimitIndicator = noLimitIndicatorV2
	} else {
		memoryLimitFile = cgroupV1MemLimitFile
		noLimitIndicator = noLimitIndicatorV1
	}

	contents, err := os.ReadFile(memoryLimitFile)
	if err != nil {
		logger.Debug().Msgf("failed to read cgroup memory limit file: %v", err)
		return -1
	}
	contentStr := strings.TrimSpace(string(contents))
	if contentStr == noLimitIndicator {
		logger.Debug().Msgf("No memory limit detected in %s; assuming unlimited", memoryLimitFile)
		return -1
	}
	maxmem, err := strconv.ParseFloat(contentStr, 64)
	if err != nil {
		logger.Debug().Msgf("Failed to parse memory limit from %s: %v", memoryLimitFile, err)
		return -1
	}
	logger.Debug().Msgf("Max memory limit: %.0f bytes", maxMemory)
	return maxmem
}

func getCPULimitFromCgroup(version CgroupVersion) int {
	var cpuQuotaFile string
	var noLimitIndicator string
	if version == CgroupV2 {
		cpuQuotaFile = cgroupV2CpuQuotaFile
		noLimitIndicator = noLimitIndicatorV2
	} else {
		cpuQuotaFile = cgroupV1CpuQuotaFile
		noLimitIndicator = noLimitIndicatorV1
	}

	contents, err := os.ReadFile(cpuQuotaFile)
	if err != nil {
		logger.Debug().Msgf("Failed to read CPU quota file %s: %v", cpuQuotaFile, err)
		return -1
	}
	contentStr := strings.TrimSpace(string(contents))
	fields := strings.Fields(contentStr)
	if len(fields) == 0 || fields[0] == noLimitIndicator {
		defaultCPULimit := runtime.NumCPU() * 1000 // Default to number of CPUs in millicores if no limit is set
		logger.Debug().Msgf("No CPU limit detected in %s; assuming %d mCPU", cpuQuotaFile, defaultCPULimit)
		return defaultCPULimit
	}
	cpuQuota, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		logger.Debug().Msgf("Failed to parse CPU quota from %s: %v", cpuQuotaFile, err)
		return -1
	}

	var cpuPeriod float64
	if version == CgroupV2 {
		cpuPeriod = getCPUPeriodCgroupV2(fields)
	} else {
		cpuPeriod = getCPUPeriodCgroupV1()
	}

	if cpuPeriod == 0 {
		logger.Debug().Msgf("CPU period is zero, cannot calculate CPU limit")
		return -1
	}
	cpuLimit := int((cpuQuota / cpuPeriod) * 1000) // Convert to millicores (100% = 1.0 = 1000 mCPU)
	logger.Debug().Msgf("CPU millicore limit from %s: %d mCPU", cpuQuotaFile, cpuLimit)
	return cpuLimit
}

func getCPUPeriodCgroupV2(fields []string) float64 {
	if len(fields) <= 1 {
		logger.Debug().Msgf("CPU period missing in %s; using default %d", cgroupV2CpuQuotaFile, defaultCgroupCPUPeriod)
		return defaultCgroupCPUPeriod
	}
	periodVal, err := strconv.ParseFloat(fields[1], 64)
	if err != nil || periodVal == 0 {
		logger.Debug().Msgf("Failed to parse CPU period from %s: %v; using default %d", cgroupV2CpuQuotaFile, err, defaultCgroupCPUPeriod)
		return defaultCgroupCPUPeriod
	}
	return periodVal
}

func getCPUPeriodCgroupV1() float64 {
	if cgroupV1CpuPeriodFile == "" {
		logger.Debug().Msgf("CPU period file not provided; using default %d", defaultCgroupCPUPeriod)
		return defaultCgroupCPUPeriod
	}
	periodContents, err := os.ReadFile(cgroupV1CpuPeriodFile)
	if err != nil {
		logger.Debug().Msgf("Failed to read CPU period file %s: %v; using default %d", cgroupV1CpuPeriodFile, err, defaultCgroupCPUPeriod)
		return defaultCgroupCPUPeriod
	}
	periodVal, err := strconv.ParseFloat(strings.TrimSpace(string(periodContents)), 64)
	if err != nil || periodVal == 0 {
		logger.Debug().Msgf("Failed to parse CPU period from %s: %v; using default %d", cgroupV1CpuPeriodFile, err, defaultCgroupCPUPeriod)
		return defaultCgroupCPUPeriod
	}
	return periodVal
}
