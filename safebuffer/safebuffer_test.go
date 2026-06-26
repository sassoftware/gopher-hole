package safebuffer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_SafeBuffer(t *testing.T) {
	sample := NewSafeBuffer(3)

	// Initially, the ring should be empty (all values nil)
	actual := sample.GetData()
	var expected []any
	assert.Equal(t, expected, actual)
	assert.Equal(t, nil, sample.GetMostRecent())

	// Add one element and check
	sample.Add(1)
	actual = sample.GetData()
	expected = []any{1}
	assert.Equal(t, expected, actual)
	assert.Equal(t, 1, sample.GetMostRecent())

	// Fill the ring and check
	sample.Add(2)
	sample.Add(3)
	actual = sample.GetData()
	expected = []any{1, 2, 3}
	assert.Equal(t, expected, actual)
	assert.Equal(t, 3, sample.GetMostRecent())

	// Add one more element to overwrite the oldest and check
	sample.Add(4)
	actual = sample.GetData()
	expected = []any{2, 3, 4}
	assert.Equal(t, expected, actual)
	assert.Equal(t, 4, sample.GetMostRecent())
}
