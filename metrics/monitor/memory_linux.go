package monitor

import (
	"errors"
	"math"
	"os"
	"runtime/metrics"
	"strconv"
	"strings"
)

var memInfoFile = "/proc/meminfo"

const goMemoryLimitMetric = "/gc/gomemlimit:bytes"

// selectMemoryLimit prefers an explicit cgroup limit, then GOMEMLIMIT, then node memory.
func selectMemoryLimit(cgroupLimit, goLimit, nodeLimit float64) float64 {
	if isFiniteMemoryLimit(cgroupLimit) {
		return cgroupLimit
	}
	if isFiniteMemoryLimit(goLimit) {
		return goLimit
	}
	if isFiniteMemoryLimit(nodeLimit) {
		return nodeLimit
	}
	return -1
}

func resolveMemoryLimit(cgroupLimit float64, getGoLimit, getNodeLimit func() float64) float64 {
	goLimit, nodeLimit := -1.0, -1.0
	if !isFiniteMemoryLimit(cgroupLimit) {
		goLimit = getGoLimit()
		if !isFiniteMemoryLimit(goLimit) {
			nodeLimit = getNodeLimit()
		}
	}
	return selectMemoryLimit(cgroupLimit, goLimit, nodeLimit)
}

func isFiniteMemoryLimit(limit float64) bool {
	return limit > 0 && !math.IsNaN(limit) && !math.IsInf(limit, 0)
}

func getGoMemoryLimit() float64 {
	sample := []metrics.Sample{{Name: goMemoryLimitMetric}}
	metrics.Read(sample)
	if sample[0].Value.Kind() != metrics.KindUint64 {
		logger.Debug().Msgf("Go runtime memory limit metric %s is unavailable", goMemoryLimitMetric)
		return -1
	}
	return normalizeGoMemoryLimit(sample[0].Value.Uint64())
}

func normalizeGoMemoryLimit(limit uint64) float64 {
	if limit == 0 || limit >= uint64(math.MaxInt64) {
		logger.Debug().Msg("Go runtime memory limit is unlimited or invalid")
		return -1
	}
	return float64(limit)
}

func getNodeMemoryLimit() float64 {
	contents, err := os.ReadFile(memInfoFile)
	if err != nil {
		logger.Debug().Msgf("failed to read node memory from %s: %v", memInfoFile, err)
		return -1
	}

	limit, err := parseNodeMemoryLimit(contents)
	if err != nil {
		logger.Debug().Msgf("failed to parse node memory from %s: %v", memInfoFile, err)
		return -1
	}
	return float64(limit)
}

func parseNodeMemoryLimit(contents []byte) (uint64, error) {
	for _, line := range strings.Split(string(contents), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != "MemTotal:" {
			continue
		}
		if len(fields) != 3 || fields[2] != "kB" {
			return 0, errors.New("unexpected MemTotal format")
		}

		kilobytes, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil || kilobytes == 0 {
			return 0, errors.New("invalid MemTotal value")
		}
		if kilobytes > ^uint64(0)/1024 {
			return 0, errors.New("MemTotal value overflows bytes")
		}
		return kilobytes * 1024, nil
	}
	return 0, errors.New("MemTotal entry not found")
}
