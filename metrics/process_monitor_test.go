//go:build !windows

package metrics

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProcessMonitorNonWindows_CancelsOnContext(t *testing.T) {
	const procMonitorRunTime = 21 * time.Second
	const postCancelWaitTime = 20 * time.Second
	synctest.Test(t, func(_ *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			ProcessMonitorNonWindows(ctx)
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
