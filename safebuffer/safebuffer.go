package safebuffer

import (
	"container/ring"
	"sync"
)

// SafeBuffer is a thread-safe circular buffer that utilizes a ring structure.
type SafeBuffer struct {
	buffer *ring.Ring
	mutex  sync.Mutex
}

// NewSafeBuffer initializes a new SafeBuffer with the specified size.
func NewSafeBuffer(size int) *SafeBuffer {
	return &SafeBuffer{
		buffer: ring.New(size),
		mutex:  sync.Mutex{},
	}
}

// Add inserts a new value into the SafeBuffer in a thread-safe manner.
func (sr *SafeBuffer) Add(val interface{}) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()
	sr.buffer.Value = val
	sr.buffer = sr.buffer.Next()
}

// GetMostRecent retrieves the most recently added entry from the SafeBuffer in a thread-safe manner.
func (sr *SafeBuffer) GetMostRecent() any {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()
	// The last added element is the one before the current position
	last := sr.buffer.Prev()
	return last.Value
}

// GetData retrieves all non-nil entries from the SafeBuffer in a thread-safe manner.
func (sr *SafeBuffer) GetData() []any {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()
	var dataSlice []any
	sr.buffer.Do(func(entry any) {
		if entry != nil {
			dataSlice = append(dataSlice, entry)
		}
	})
	return dataSlice
}
