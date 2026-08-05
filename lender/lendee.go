package lender

import (
	"context"
	"fmt"
	"sync/atomic"
)

// BaseLendee provides a default implementation of the Lendee interface.
// It manages processing credits using a buffered channel. Consumers can embed
// this struct to inherit all credit management functionality without having to
// re-implement the Lendee interface.
type BaseLendee struct {
	name string
	// credits is a buffered channel used as a semaphore for controlling
	// concurrent processing capacity. Channel capacity = maxTotalCredits.
	credits chan struct{}
	// initialCredits is the number of credits primed at startup.
	// The Lender cannot take these away.
	initialCredits int
	// allocatedCredits tracks the number of credits currently lent to this lendee
	// by a Lender (via AddCredit/RemoveCredit). This value is always <= maxLenderCredits.
	// Uses atomic operations for thread safety.
	allocatedCredits atomic.Int64
	// heldCredits tracks the number of credits currently acquired by borrowers
	// but not yet released. Used to prevent over-releasing back into the channel.
	// Uses atomic operations for thread safety.
	heldCredits atomic.Int64
}

// NewBaseLendee creates a new BaseLendee with the given name, initial credits,
// and max lender credits.
func NewBaseLendee(name string, initialCredits int, maxTotalCredits int) (*BaseLendee, error) {
	if maxTotalCredits < 1 {
		return nil, fmt.Errorf("maxTotalCredits must be at least 1")
	}
	if initialCredits < 0 {
		return nil, fmt.Errorf("initialCredits cannot be negative")
	}
	if initialCredits > maxTotalCredits {
		return nil, fmt.Errorf("initialCredits cannot exceed maxTotalCredits")
	}

	bl := &BaseLendee{
		name:           name,
		credits:        make(chan struct{}, maxTotalCredits),
		initialCredits: initialCredits,
	}

	// Fill the channel with initialCredits.
	for i := 0; i < initialCredits; i++ {
		bl.credits <- struct{}{}
	}

	return bl, nil
}

// AddCredit adds a processing credit lent by a Lender, increasing capacity by one.
// Returns an error if the allocated credits would exceed maxLenderCredits.
func (bl *BaseLendee) AddCredit() error {
	// Atomically claim the right to add one lender credit by incrementing first.
	if bl.allocatedCredits.Add(1) > int64(bl.MaxLenderCredits()) {
		bl.allocatedCredits.Add(-1)
		return fmt.Errorf("cannot add credit: at maxLenderCredits (%d) for lendee %s", bl.MaxLenderCredits(), bl.name)
	}
	err := bl.giveCredit()
	if err != nil {
		// If we failed to release the credit, roll back the counter increment.
		bl.allocatedCredits.Add(-1)
		return err
	}
	return nil
}

// RemoveCredit removes a processing credit previously lent by a Lender,
// decreasing capacity by one.
func (bl *BaseLendee) RemoveCredit(ctx context.Context) error {
	// Atomically claim the right to remove one lender credit by decrementing
	// first.
	if bl.allocatedCredits.Add(-1) < 0 {
		bl.allocatedCredits.Add(1)
		return fmt.Errorf("no credit allocated, cannot remove credit")
	}

	if err := bl.takeCredit(ctx); err != nil {
		// Channel was closed or context was canceled; restore the counter.
		bl.allocatedCredits.Add(1)
		return err
	}
	return nil
}

// MaxCredits returns the total channel capacity (initialCredits + maxLenderCredits).
func (bl *BaseLendee) maxCredits() int {
	return cap(bl.credits)
}

// MaxLenderCredits returns the maximum number of credits the Lender can add.
func (bl *BaseLendee) MaxLenderCredits() int {
	return bl.maxCredits() - bl.initialCredits
}

// availableCredits returns the number of credits currently available for processing.
func (bl *BaseLendee) availableCredits() int {
	return len(bl.credits)
}

// currentCredits returns the total number of credits currently in the system,
// including both available and lent credits.
func (bl *BaseLendee) currentCredits() int {
	return bl.initialCredits + int(bl.allocatedCredits.Load())
}

// Availability returns the percentage of credits that are currently available for use, as a value between 0 and 1.
func (bl *BaseLendee) Availability() float64 {
	if bl.currentCredits() == 0 {
		return 0.0
	}
	return float64(bl.availableCredits()) / float64(bl.currentCredits())
}

// GetName returns the name of the lendee.
func (bl *BaseLendee) GetName() string {
	return bl.name
}

// AcquireCredit blocks until a processing credit is available, the context is
// canceled, or the credit channel is closed.
func (bl *BaseLendee) AcquireCredit(ctx context.Context) error {
	if err := bl.takeCredit(ctx); err != nil {
		return err
	}
	bl.heldCredits.Add(1)
	return nil
}

// ReleaseCredit returns a processing credit to the pool.
// Returns an error if no credit is currently held or the channel is full.
func (bl *BaseLendee) ReleaseCredit() error {
	if bl.heldCredits.Add(-1) < 0 {
		bl.heldCredits.Add(1)
		return fmt.Errorf("no acquired credit to release for lendee %s", bl.name)
	}
	if err := bl.giveCredit(); err != nil {
		bl.heldCredits.Add(1)
		return err
	}
	return nil
}

// CloseCredits closes the credit channel. This should only be called during shutdown.
func (bl *BaseLendee) CloseCredits() {
	close(bl.credits)
}

func (bl *BaseLendee) takeCredit(ctx context.Context) error {
	select {
	case _, ok := <-bl.credits:
		if !ok {
			return fmt.Errorf("credit channel closed while waiting for a credit")
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled while waiting to acquire credit for lendee %s: %w", bl.name, ctx.Err())
	}
}

func (bl *BaseLendee) giveCredit() error {
	select {
	case bl.credits <- struct{}{}:
		return nil
	default:
		return fmt.Errorf("credit channel full when trying to release credit for lendee %s", bl.name)
	}
}
