package lender

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBaseLendee(t *testing.T) {
	bl, err := NewBaseLendee("test-lendee", 5, 8)
	require.NoError(t, err)

	assert.Equal(t, "test-lendee", bl.GetName())
	assert.Equal(t, 8, bl.maxCredits())                 // passed in
	assert.Equal(t, 5, bl.initialCredits)               // initialCredits
	assert.Equal(t, 3, bl.MaxLenderCredits())           // maxLenderCredits
	assert.Equal(t, 5, bl.availableCredits())           // starts with initialCredits
	assert.Equal(t, 0, int(bl.allocatedCredits.Load())) // no credits given from lender
	assert.Equal(t, 5, bl.currentCredits())             // preloaded with 5 credits
}

func TestBaseLendee_AddCredit(t *testing.T) {
	bl, err := NewBaseLendee("test-lendee", 3, 5)
	require.NoError(t, err)

	// Add a credit from lender (should work, channel has room)
	err = bl.AddCredit()
	assert.NoError(t, err)
	assert.Equal(t, 4, bl.availableCredits()) // 3 initial + 1 added
	assert.Equal(t, 1, int(bl.allocatedCredits.Load()))
}

func TestBaseLendee_AddCredit_AtMaxLenderCredits(t *testing.T) {
	bl, err := NewBaseLendee("test-lendee", 3, 5)
	require.NoError(t, err)

	// Add 2 credits (maxLenderCredits)
	err = bl.AddCredit()
	assert.NoError(t, err)
	err = bl.AddCredit()
	assert.NoError(t, err)
	assert.Equal(t, 2, int(bl.allocatedCredits.Load()))

	// Adding one more should fail because we've hit maxLenderCredits
	err = bl.AddCredit()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maxLenderCredits")
	assert.Equal(t, 2, int(bl.allocatedCredits.Load()))
}

func TestBaseLendee_RemoveCredit(t *testing.T) {
	bl, err := NewBaseLendee("test-lendee", 3, 5)
	require.NoError(t, err)

	// Add 2 credits from lender
	err = bl.AddCredit()
	assert.NoError(t, err)
	err = bl.AddCredit()
	assert.NoError(t, err)
	assert.Equal(t, 5, bl.availableCredits()) // 3 initial + 2 added
	assert.Equal(t, 2, int(bl.allocatedCredits.Load()))

	// Remove a credit
	err = bl.RemoveCredit(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 4, bl.availableCredits())
	assert.Equal(t, 1, int(bl.allocatedCredits.Load()))

	// Remove another credit
	err = bl.RemoveCredit(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 3, bl.availableCredits())
	assert.Equal(t, 0, int(bl.allocatedCredits.Load()))

	// Removing more should not reduce allocatedCredits below 0
	err = bl.RemoveCredit(context.Background())
	assert.Error(t, err)
	assert.Equal(t, 3, bl.availableCredits())           // still 3 initial credits available
	assert.Equal(t, 0, int(bl.allocatedCredits.Load())) // cannot go below 0
}

func TestBaseLendee_AddRemoveCredit_AllocatedTracking(t *testing.T) {
	bl, err := NewBaseLendee("test-lendee", 3, 6)
	require.NoError(t, err)

	// Add two credits from a Lender
	_ = bl.AddCredit()
	_ = bl.AddCredit()
	assert.Equal(t, 5, bl.availableCredits()) // 3 initial + 2 added
	assert.Equal(t, 2, int(bl.allocatedCredits.Load()))

	// Remove one lent credit
	_ = bl.RemoveCredit(context.Background())
	assert.Equal(t, 4, bl.availableCredits())
	assert.Equal(t, 1, int(bl.allocatedCredits.Load()))
}

func TestBaseLendee_ImplementsLendeeInterface(t *testing.T) {
	bl, err := NewBaseLendee("test-lendee", 5, 8)
	require.NoError(t, err)
	var lendee Lendee = bl

	assert.Equal(t, "test-lendee", lendee.GetName())
	assert.Equal(t, 3, lendee.MaxLenderCredits()) // passed in
	assert.Equal(t, 1.0, lendee.Availability())
}

func TestBaseLendee_MaxCreditsFormula(t *testing.T) {
	const initialCredits = 5
	const maxTotalCredits = 10
	bl, err := NewBaseLendee("test-lendee", initialCredits, maxTotalCredits)
	require.NoError(t, err)

	// Verify the formula
	assert.Equal(t, maxTotalCredits, bl.maxCredits())
	assert.Equal(t, initialCredits, bl.initialCredits)
	assert.Equal(t, maxTotalCredits-initialCredits, bl.MaxLenderCredits())

	// Verify channel is primed with initialCredits
	assert.Equal(t, initialCredits, bl.availableCredits())

	// Lender can add up to maxLenderCredits
	for i := 0; i < bl.MaxLenderCredits(); i++ {
		err := bl.AddCredit()
		assert.NoError(t, err)
	}
	assert.Equal(t, bl.MaxLenderCredits(), int(bl.allocatedCredits.Load()))

	// Channel should now be at maxTotalCredits capacity
	assert.Equal(t, maxTotalCredits, bl.availableCredits())

	// Cannot add more credits - at maxLenderCredits limit
	err = bl.AddCredit()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maxLenderCredits")
}

func TestBaseLendee_AcquireAndReleaseCredit(t *testing.T) {
	bl, err := NewBaseLendee("test-lendee", 2, 4)
	require.NoError(t, err)
	assert.Equal(t, 2, bl.availableCredits())

	// Acquire a credit
	err = bl.AcquireCredit(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, bl.availableCredits())

	// Acquire another credit
	err = bl.AcquireCredit(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 0, bl.availableCredits())

	// Release a credit
	err = bl.ReleaseCredit()
	assert.NoError(t, err)
	assert.Equal(t, 1, bl.availableCredits())

	// Release another credit
	err = bl.ReleaseCredit()
	assert.NoError(t, err)
	assert.Equal(t, 2, bl.availableCredits())
}

func TestBaseLendee_AcquireCredit_BlocksWhenEmpty(t *testing.T) {
	bl, err := NewBaseLendee("test-lendee", 1, 2)
	require.NoError(t, err)

	// Acquire the only credit
	err = bl.AcquireCredit(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 0, bl.availableCredits())

	// AcquireCredit should block when no credits available
	acquired := make(chan bool)
	go func() {
		_ = bl.AcquireCredit(context.Background())
		acquired <- true
	}()

	// Should not complete immediately
	select {
	case <-acquired:
		t.Fatal("AcquireCredit should have blocked when no credits available")
	case <-time.After(100 * time.Millisecond):
		// Expected - blocked
	}

	// Release a credit so AcquireCredit can proceed
	_ = bl.ReleaseCredit()

	select {
	case <-acquired:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("AcquireCredit should have completed after ReleaseCredit")
	}
}

func TestBaseLendee_CloseCredits(t *testing.T) {
	bl, err := NewBaseLendee("test-lendee", 2, 4)
	require.NoError(t, err)

	// Acquire credits to make them blocked
	waiting := make(chan bool)
	go func() {
		_ = bl.AcquireCredit(context.Background())
		_ = bl.AcquireCredit(context.Background())
		err := bl.AcquireCredit(context.Background()) // This will block
		if err != nil {
			waiting <- true // Error means channel was closed
		}
	}()

	// Give time for the goroutine to block on AcquireCredit
	time.Sleep(50 * time.Millisecond)

	// Close the credits channel
	bl.CloseCredits()

	// The blocked AcquireCredit should return with an error
	select {
	case <-waiting:
		// Expected - closed channel caused AcquireCredit to return error
	case <-time.After(200 * time.Millisecond):
		t.Fatal("CloseCredits should have unblocked AcquireCredit")
	}
}

func TestBaseLendee_LenderCannotTakeInitialCredits(t *testing.T) {
	bl, err := NewBaseLendee("test-lendee", 3, 5)
	require.NoError(t, err)

	// Add 2 lender credits
	err = bl.AddCredit()
	assert.NoError(t, err)
	err = bl.AddCredit()
	assert.NoError(t, err)
	assert.Equal(t, 2, int(bl.allocatedCredits.Load()))
	assert.Equal(t, 5, bl.availableCredits()) // 3 initial + 2 lender

	// Remove lender credits
	err = bl.RemoveCredit(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, int(bl.allocatedCredits.Load()))
	err = bl.RemoveCredit(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 0, int(bl.allocatedCredits.Load()))

	// Even if we remove more, allocatedCredits stays at 0
	// (lender can't take initial credits)
	err = bl.RemoveCredit(context.Background())
	assert.Error(t, err)
	assert.Equal(t, 0, int(bl.allocatedCredits.Load()))
	assert.Equal(t, 3, bl.availableCredits())
}

func TestBaseLendee_Concurrency(t *testing.T) {
	tests := map[string]struct {
		initialCredits  int
		maxTotalCredits int
		setup           func(t *testing.T, bl *BaseLendee)
		timeout         time.Duration
	}{
		"no lender credits, only using 5 initial credits": {
			initialCredits:  5,
			maxTotalCredits: 10,
			setup:           nil,
		},
		"max lender credits, using 10 total credits": {
			initialCredits:  5,
			maxTotalCredits: 10,
			setup: func(t *testing.T, bl *BaseLendee) {
				for i := 0; i < bl.MaxLenderCredits(); i++ {
					err := bl.AddCredit()
					require.NoError(t, err)
				}
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			bl, err := NewBaseLendee("test-lendee", 5, 10)
			require.NoError(t, err)

			if tc.setup != nil {
				tc.setup(t, bl)
			}

			const numProcesses = 20
			const sleepTimePerProcess = 100 * time.Millisecond
			creditsToUse := bl.availableCredits()
			timeout := sleepTimePerProcess * time.Duration((numProcesses/creditsToUse)+2) // Enough time for all processes to acquire and release credits
			ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
			defer cancelFunc()
			var processed atomic.Int64
			for i := 0; i < numProcesses; i++ {
				err = bl.AcquireCredit(ctx)
				assert.NoError(t, err)
				go func() {
					defer func() {
						releaseErr := bl.ReleaseCredit()
						assert.NoError(t, releaseErr)
						processed.Add(1)
					}()
					time.Sleep(sleepTimePerProcess) // Simulate work
				}()
			}
			<-ctx.Done()
			assert.Equal(t, creditsToUse, bl.availableCredits())
			assert.Equal(t, numProcesses, int(processed.Load()))
		})
	}
}
