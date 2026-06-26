package math

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoundFloat(t *testing.T) {
	tests := map[string]struct {
		val       float64
		precision uint
		expected  float64
	}{
		"round to 0 decimal": {
			val:       1.4,
			precision: 0,
			expected:  1,
		},
		"round up to 0 decimal": {
			val:       1.5,
			precision: 0,
			expected:  2,
		},
		"round to 1 decimal": {
			val:       1.44,
			precision: 1,
			expected:  1.4,
		},
		"round up to 1 decimal": {
			val:       1.45,
			precision: 1,
			expected:  1.5,
		},
		"round negative number": {
			val:       -1.45,
			precision: 1,
			expected:  -1.5,
		},
		"round zero": {
			val:       0,
			precision: 2,
			expected:  0,
		},
		"round large precision": {
			val:       1.23456789,
			precision: 6,
			expected:  1.234568,
		},
		"round negative precision": {
			val:       -1.23456789,
			precision: 6,
			expected:  -1.234568,
		},
		"round large number": {
			val:       123456.789,
			precision: 2,
			expected:  123456.79,
		},
		"round small number": {
			val:       0.000123456,
			precision: 7,
			expected:  0.0001235,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := RoundFloat(tt.val, tt.precision)
			assert.Equal(t, tt.expected, result)
		})
	}
}
