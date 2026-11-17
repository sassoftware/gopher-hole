package monitor

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	runtimeMetrics "runtime/metrics"

	"github.com/sassoftware/gopher-hole/util/safebuffer"
	"github.com/stretchr/testify/assert"
)

func TestSelectMemoryLimit(t *testing.T) {
	const (
		gib = float64(1024 * 1024 * 1024)
	)
	tests := []struct {
		name      string
		cgroup    float64
		goLimit   float64
		nodeLimit float64
		expected  float64
	}{
		{
			name:      "finite cgroup limit preserves current behavior",
			cgroup:    8 * gib,
			goLimit:   4 * gib,
			nodeLimit: 16 * gib,
			expected:  8 * gib,
		},
		{
			name:      "prefer Go memory limit when node memory is larger",
			cgroup:    -1,
			goLimit:   4 * gib,
			nodeLimit: 16 * gib,
			expected:  4 * gib,
		},
		{
			name:      "prefer Go memory limit when node memory is smaller",
			cgroup:    -1,
			goLimit:   16 * gib,
			nodeLimit: 4 * gib,
			expected:  16 * gib,
		},
		{
			name:      "node memory is fallback when Go memory limit is unlimited",
			cgroup:    -1,
			goLimit:   normalizeGoMemoryLimit(uint64(math.MaxInt64)),
			nodeLimit: 16 * gib,
			expected:  16 * gib,
		},
		{
			name:      "Go limit remains usable when node memory cannot be read",
			cgroup:    -1,
			goLimit:   4 * gib,
			nodeLimit: -1,
			expected:  4 * gib,
		},
		{
			name:      "no trustworthy source keeps limit unknown",
			cgroup:    -1,
			goLimit:   -1,
			nodeLimit: -1,
			expected:  -1,
		},
		{
			name:      "invalid Go limit falls back to node memory",
			cgroup:    math.NaN(),
			goLimit:   math.Inf(1),
			nodeLimit: 8 * gib,
			expected:  8 * gib,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, selectMemoryLimit(tt.cgroup, tt.goLimit, tt.nodeLimit))
		})
	}
}

func TestNormalizeGoMemoryLimit(t *testing.T) {
	const gib = uint64(1024 * 1024 * 1024)
	assert.Equal(t, float64(gib), normalizeGoMemoryLimit(gib))
	assert.Equal(t, float64(-1), normalizeGoMemoryLimit(0))
	assert.Equal(t, float64(-1), normalizeGoMemoryLimit(uint64(math.MaxInt64)))
}

func TestParseNodeMemoryLimit(t *testing.T) {
	contents := []byte("MemTotal:       16384 kB\nMemFree:         4096 kB\n")
	limit, err := parseNodeMemoryLimit(contents)
	assert.NoError(t, err)
	assert.Equal(t, uint64(16*1024*1024), limit)

	for _, invalid := range [][]byte{
		[]byte("MemFree: 4096 kB\n"),
		[]byte("MemTotal: invalid kB\n"),
		[]byte("MemTotal: 0 kB\n"),
		[]byte("MemTotal: 4096 MB\n"),
		[]byte("MemTotal: 18446744073709551615 kB\n"),
	} {
		_, err := parseNodeMemoryLimit(invalid)
		assert.Error(t, err)
	}
}

func TestGetNodeMemoryLimit(t *testing.T) {
	originalMemInfoFile := memInfoFile
	t.Cleanup(func() { memInfoFile = originalMemInfoFile })

	memInfoFile = filepath.Join(t.TempDir(), "meminfo")
	err := os.WriteFile(memInfoFile, []byte("MemTotal: 8192 kB\n"), 0600)
	assert.NoError(t, err)
	assert.Equal(t, float64(8*1024*1024), getNodeMemoryLimit())

	memInfoFile = filepath.Join(t.TempDir(), "missing-meminfo")
	assert.Equal(t, float64(-1), getNodeMemoryLimit())
}

func TestMemoryAvailabilityWithSelectedFallback(t *testing.T) {
	originalMemHistory := memHistory
	originalMaxMemory := maxMemory
	t.Cleanup(func() {
		memHistory = originalMemHistory
		maxMemory = originalMaxMemory
	})

	memHistory = safebuffer.New(1)
	memHistory.Add(uint64(2 * 1024 * 1024 * 1024))
	maxMemory = selectMemoryLimit(-1, float64(8*1024*1024*1024), float64(16*1024*1024*1024))
	assert.Equal(t, 0.75, GetMemoryAvailability())

	memHistory = safebuffer.New(1)
	memHistory.Add(uint64(2 * 1024 * 1024 * 1024))
	maxMemory = selectMemoryLimit(-1, -1, -1)
	assert.Equal(t, -1.0, GetMemoryAvailability())
}

func TestGetGoMemoryLimit(t *testing.T) {
	sample := []runtimeMetrics.Sample{{Name: goMemoryLimitMetric}}
	runtimeMetrics.Read(sample)
	assert.Equal(t, runtimeMetrics.KindUint64, sample[0].Value.Kind())
	assert.Equal(t, normalizeGoMemoryLimit(sample[0].Value.Uint64()), getGoMemoryLimit())
}
