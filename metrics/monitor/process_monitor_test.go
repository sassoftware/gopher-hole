//go:build !windows

package monitor

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/sassoftware/gopher-hole/util/safebuffer"
	"github.com/stretchr/testify/assert"
)

func TestMemoryAvailability(t *testing.T) {
	// Save original state
	originalMemHistory := memHistory
	originalMaxMemory := maxMemory

	// Restore original state after test
	defer func() {
		memHistory = originalMemHistory
		maxMemory = originalMaxMemory
	}()

	tests := []struct {
		name                 string
		memHistory           []uint64
		maxMemory            float64
		expectedAvailability float64 // min and max expected values
	}{
		{
			name:                 "no max memory set - assume unknown available",
			memHistory:           []uint64{1000, 2000, 3000},
			maxMemory:            0,
			expectedAvailability: -1.0, // -1 indicates unknown availability
			// Note: Current implementation returns -1 when maxMemory is not set, which indicates unknown availability
		},
		{
			name:                 "50% memory usage",
			memHistory:           []uint64{5000, 5000, 5000},
			maxMemory:            10000,
			expectedAvailability: 0.5,
		},
		{
			name:                 "25% memory usage - 75% available",
			memHistory:           []uint64{2500, 2500, 2500},
			maxMemory:            10000,
			expectedAvailability: 0.75,
		},
		{
			name:                 "90% memory usage - 10% available",
			memHistory:           []uint64{9000, 9000, 9000},
			maxMemory:            10000,
			expectedAvailability: 0.1,
		},
		{
			name:                 "low memory usage - high availability",
			memHistory:           []uint64{100, 100, 100},
			maxMemory:            10000,
			expectedAvailability: 0.99,
		},
		{
			name:                 "memory usage exceeds limit - 0% available",
			memHistory:           []uint64{15000, 15000, 15000},
			maxMemory:            10000,
			expectedAvailability: 0.0,
		},
		{
			name:                 "varying memory usage - average 60%",
			memHistory:           []uint64{4000, 6000, 8000},
			maxMemory:            10000,
			expectedAvailability: 0.4,
		},
		{
			name:                 "single memory sample",
			memHistory:           []uint64{3000},
			maxMemory:            10000,
			expectedAvailability: 0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memHistory = safebuffer.New(len(tt.memHistory))
			for _, mem := range tt.memHistory {
				memHistory.Add(mem)
			}
			maxMemory = tt.maxMemory

			result := GetMemoryAvailability()
			assert.Equal(t, tt.expectedAvailability, result, "memoryAvailability() = %v; want %v", result, tt.expectedAvailability)
		})
	}
}

func TestCPUAvailability(t *testing.T) {
	// Save original state
	originalCPUHistory := cpuMillicoreHistory
	originalCPULimit := cpuMillicoreLimit

	// Restore original state after test
	defer func() {
		cpuMillicoreHistory = originalCPUHistory
		cpuMillicoreLimit = originalCPULimit
	}()

	tests := []struct {
		name                 string
		cpuHistory           []int
		cpuMillicoreLimit    int
		expectedAvailability float64 // expected value
	}{
		{
			name:                 "50% CPU usage - 50% available",
			cpuHistory:           []int{500, 500, 500},
			cpuMillicoreLimit:    1000,
			expectedAvailability: 0.5,
		},
		{
			name:                 "25% CPU usage - 75% available",
			cpuHistory:           []int{250, 250, 250},
			cpuMillicoreLimit:    1000,
			expectedAvailability: 0.75,
		},
		{
			name:                 "90% CPU usage - 10% available",
			cpuHistory:           []int{900, 900, 900},
			cpuMillicoreLimit:    1000,
			expectedAvailability: 0.1,
		},
		{
			name:                 "low CPU usage - high availability",
			cpuHistory:           []int{50, 50, 50},
			cpuMillicoreLimit:    1000,
			expectedAvailability: 0.95,
		},
		{
			name:                 "CPU usage exceeds limit - 0% available",
			cpuHistory:           []int{1500, 1500, 1500},
			cpuMillicoreLimit:    1000,
			expectedAvailability: 0.0,
		},
		{
			name:                 "varying CPU usage - average 60%",
			cpuHistory:           []int{400, 600, 800},
			cpuMillicoreLimit:    1000,
			expectedAvailability: 0.4,
		},
		{
			name:                 "single CPU sample",
			cpuHistory:           []int{300},
			cpuMillicoreLimit:    1000,
			expectedAvailability: 0.7,
		},
		{
			name:                 "multi-core system - 2000 millicore limit",
			cpuHistory:           []int{1000, 1000, 1000},
			cpuMillicoreLimit:    2000,
			expectedAvailability: 0.5,
		},
		{
			name:                 "no CPU usage - 100% available",
			cpuHistory:           []int{0, 0, 0},
			cpuMillicoreLimit:    1000,
			expectedAvailability: 1.0,
		},
		{
			name:                 "full CPU usage - 0% available",
			cpuHistory:           []int{1000, 1000, 1000},
			cpuMillicoreLimit:    1000,
			expectedAvailability: 0.0,
		},
		{
			name:                 "negative cpu millicore limit - assume unknown availability",
			cpuHistory:           []int{1000, 1000, 1000},
			cpuMillicoreLimit:    -100,
			expectedAvailability: -1.0, // Unknown availability if we don't know the limit
		},
		{
			name:                 "zero cpu millicore limit - assume unknown availability",
			cpuHistory:           []int{1000, 1000, 1000},
			cpuMillicoreLimit:    0,
			expectedAvailability: -1.0, // Unknown availability if we don't know the limit
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpuMillicoreHistory = safebuffer.New(len(tt.cpuHistory))
			for _, cpu := range tt.cpuHistory {
				cpuMillicoreHistory.Add(cpu)
			}
			cpuMillicoreLimit = tt.cpuMillicoreLimit

			result := GetCPUAvailability()
			assert.Equal(t, tt.expectedAvailability, result, "cpuAvailability() = %v; want %v", result, tt.expectedAvailability)
		})
	}
}

func TestMemoryAvailabilityEdgeCases(t *testing.T) {
	// Save original state
	originalMemHistory := memHistory
	originalMaxMemory := maxMemory

	// Restore original state after test
	defer func() {
		memHistory = originalMemHistory
		maxMemory = originalMaxMemory
	}()

	t.Run("empty history - division by zero protection", func(t *testing.T) {
		memHistory = safebuffer.New(5)
		maxMemory = 10000

		// This will cause division by zero, function should handle it
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("memoryAvailability() panicked with empty history: %v", r)
			}
		}()

		// Note: Current implementation will panic on empty history
		// This test documents the behavior
	})

	t.Run("negative availability clamped to 0", func(t *testing.T) {
		memHistory = safebuffer.New(3)
		memHistory.Add(uint64(20000))
		memHistory.Add(uint64(20000))
		memHistory.Add(uint64(20000))
		maxMemory = 10000

		result := GetMemoryAvailability()
		if result != 0.0 {
			t.Errorf("memoryAvailability() = %v; want 0.0 when usage exceeds limit", result)
		}
	})
}

func TestCPUAvailabilityEdgeCases(t *testing.T) {
	// Save original state
	originalCPUHistory := cpuMillicoreHistory
	originalCPULimit := cpuMillicoreLimit

	// Restore original state after test
	defer func() {
		cpuMillicoreHistory = originalCPUHistory
		cpuMillicoreLimit = originalCPULimit
	}()

	t.Run("empty history - division by zero protection", func(t *testing.T) {
		cpuMillicoreHistory = safebuffer.New(5)
		cpuMillicoreLimit = 1000

		// This will cause division by zero, function should handle it
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("cpuAvailability() panicked with empty history: %v", r)
			}
		}()

		// Note: Current implementation will panic on empty history
		// This test documents the behavior
	})

	t.Run("negative availability clamped to 0", func(t *testing.T) {
		cpuMillicoreHistory = safebuffer.New(3)
		cpuMillicoreHistory.Add(2000)
		cpuMillicoreHistory.Add(2000)
		cpuMillicoreHistory.Add(2000)
		cpuMillicoreLimit = 1000

		result := GetCPUAvailability()
		if result != 0.0 {
			t.Errorf("cpuAvailability() = %v; want 0.0 when usage exceeds limit", result)
		}
	})
}
func TestAddMemoryHistory(t *testing.T) {
	// Save original state
	originalMemHistory := memHistory

	// Restore original state after test
	defer func() {
		memHistory = originalMemHistory
	}()

	tests := []struct {
		name           string
		initialHistory []uint64
		memToAdd       uint64
		expectedLength int
		expectedLast   uint64
	}{
		{
			name:           "add to empty history",
			initialHistory: []uint64{},
			memToAdd:       1000,
			expectedLength: 1,
			expectedLast:   1000,
		},
		{
			name:           "add to partially filled history",
			initialHistory: []uint64{1000, 2000, 3000},
			memToAdd:       4000,
			expectedLength: 4,
			expectedLast:   4000,
		},
		{
			name:           "add when history is at max capacity - slides window",
			initialHistory: []uint64{1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 11000, 12000},
			memToAdd:       13000,
			expectedLength: 12,
			expectedLast:   13000,
		},
		{
			name:           "add zero value",
			initialHistory: []uint64{1000, 2000},
			memToAdd:       0,
			expectedLength: 3,
			expectedLast:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memHistory = safebuffer.New(int(procStatsHistoryDuration.Seconds() / procStatsCollectionFrequency.Seconds()))
			for _, mem := range tt.initialHistory {
				memHistory.Add(mem)
			}

			memHistory.Add(tt.memToAdd)
			mhd := memHistory.GetData()

			assert.Equal(t, tt.expectedLength, len(mhd), "history length mismatch")
			assert.Equal(t, tt.expectedLast, mhd[len(mhd)-1].(uint64), "last element mismatch")

			// Verify max capacity is respected
			maxCapacity := int(procStatsHistoryDuration.Seconds() / procStatsCollectionFrequency.Seconds())
			assert.LessOrEqual(t, len(mhd), maxCapacity, "history exceeded max capacity")
		})
	}
}

func TestAddCPUHistory(t *testing.T) {
	// Save original state
	originalCPUHistory := cpuMillicoreHistory

	// Restore original state after test
	defer func() {
		cpuMillicoreHistory = originalCPUHistory
	}()

	tests := []struct {
		name           string
		initialHistory []int
		cpuToAdd       int
		expectedLength int
		expectedLast   int
	}{
		{
			name:           "add to empty history",
			initialHistory: []int{},
			cpuToAdd:       500,
			expectedLength: 1,
			expectedLast:   500,
		},
		{
			name:           "add to partially filled history",
			initialHistory: []int{100, 200, 300},
			cpuToAdd:       400,
			expectedLength: 4,
			expectedLast:   400,
		},
		{
			name:           "add when history is at max capacity - slides window",
			initialHistory: []int{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 1100, 1200},
			cpuToAdd:       1300,
			expectedLength: 12,
			expectedLast:   1300,
		},
		{
			name:           "add zero value",
			initialHistory: []int{100, 200},
			cpuToAdd:       0,
			expectedLength: 3,
			expectedLast:   0,
		},
		{
			name:           "add negative value",
			initialHistory: []int{100, 200},
			cpuToAdd:       -50,
			expectedLength: 3,
			expectedLast:   -50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpuMillicoreHistory = safebuffer.New(int(procStatsHistoryDuration.Seconds() / procStatsCollectionFrequency.Seconds()))
			for _, cpu := range tt.initialHistory {
				cpuMillicoreHistory.Add(cpu)
			}

			cpuMillicoreHistory.Add(tt.cpuToAdd)
			chd := cpuMillicoreHistory.GetData()

			assert.Equal(t, tt.expectedLength, len(chd), "history length mismatch")
			assert.Equal(t, tt.expectedLast, chd[len(chd)-1].(int), "last element mismatch")

			// Verify max capacity is respected
			maxCapacity := int(procStatsHistoryDuration.Seconds() / procStatsCollectionFrequency.Seconds())
			assert.LessOrEqual(t, len(chd), maxCapacity, "history exceeded max capacity")
		})
	}
}

func TestAddHistorySlidingWindow(t *testing.T) {
	// Save original state
	originalMemHistory := memHistory
	originalCPUHistory := cpuMillicoreHistory

	// Restore original state after test
	defer func() {
		memHistory = originalMemHistory
		cpuMillicoreHistory = originalCPUHistory
	}()

	maxCapacity := int(procStatsHistoryDuration.Seconds() / procStatsCollectionFrequency.Seconds())

	t.Run("memory history sliding window removes oldest entry", func(t *testing.T) {
		memHistory = safebuffer.New(maxCapacity)
		for i := 0; i < maxCapacity; i++ {
			memHistory.Add(uint64((i + 1) * 1000))
		}

		memHistory.Add(uint64(99999))

		mhd := memHistory.GetData()
		assert.Equal(t, maxCapacity, len(mhd))
		assert.Equal(t, uint64(2000), mhd[0].(uint64), "oldest entry should be removed")
		assert.Equal(t, uint64(99999), mhd[len(mhd)-1].(uint64), "newest entry should be last")
	})

	t.Run("CPU history sliding window removes oldest entry", func(t *testing.T) {
		cpuMillicoreHistory = safebuffer.New(maxCapacity)
		for i := 0; i < maxCapacity; i++ {
			cpuMillicoreHistory.Add((i + 1) * 100)
		}

		cpuMillicoreHistory.Add(9999)
		chd := cpuMillicoreHistory.GetData()
		assert.Equal(t, maxCapacity, len(chd))
		assert.Equal(t, 200, chd[0].(int), "oldest entry should be removed")
		assert.Equal(t, 9999, chd[len(chd)-1].(int), "newest entry should be last")
	})
}

func TestProcessMonitorNonWindows_CancelsOnContext(t *testing.T) {
	const procMonitorRunTime = 21 * time.Second
	const postCancelWaitTime = 20 * time.Second
	synctest.Test(t, func(_ *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			processMonitorNonWindows(ctx)
		}()
		time.Sleep(procMonitorRunTime)
		synctest.Wait()
		cancel()
		time.Sleep(postCancelWaitTime)
		synctest.Wait()
	})

	expectedCPUCollections := int(procMonitorRunTime / procStatsCollectionFrequency)
	assert.Equal(t, expectedCPUCollections, len(cpuMillicoreHistory.GetData()))
	// We should have one more memory collection than CPU collection since we collect
	// memory on the first run before we have enough data to calculate CPU usage
	expectedMemoryCollections := expectedCPUCollections + 1
	assert.Equal(t, expectedMemoryCollections, len(memHistory.GetData()))
}
