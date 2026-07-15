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

// New initializes a new SafeBuffer with the specified size.
func New(size int) *SafeBuffer {
	return &SafeBuffer{
		buffer: ring.New(size),
		mutex:  sync.Mutex{},
	}
}

// Add inserts a new value into the SafeBuffer in a thread-safe manner.
func (sb *SafeBuffer) Add(val interface{}) {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()
	sb.buffer.Value = val
	sb.buffer = sb.buffer.Next()
}

// GetMostRecent retrieves the most recently added entry from the SafeBuffer in a thread-safe manner.
func (sb *SafeBuffer) GetMostRecent() any {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()
	// The last added element is the one before the current position
	last := sb.buffer.Prev()
	return last.Value
}

// GetData retrieves all non-nil entries from the SafeBuffer in a thread-safe manner.
func (sb *SafeBuffer) GetData() []any {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()
	var dataSlice []any
	sb.buffer.Do(func(entry any) {
		if entry != nil {
			dataSlice = append(dataSlice, entry)
		}
	})
	return dataSlice
}
