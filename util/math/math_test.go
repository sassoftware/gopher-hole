package math

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const defaultRecordsSlope = 1.1714285714285715

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

func Test_calculateLinearRegressionSlope(t *testing.T) {
	tests := map[string]struct {
		dataPoints []DataPoint
		expected   float64
		expectErr  bool
	}{
		"no dataPoints": {
			dataPoints: []DataPoint{},
			expectErr:  true,
		},
		"single dataPoint": {
			dataPoints: []DataPoint{
				{0.0, 10.0},
			},
			expectErr: true,
		},
		"zero denominator": {
			dataPoints: []DataPoint{
				{1.0, 10.0},
				{1.0, 11.0},
				{1.0, 12.0},
			},
			expectErr: true,
		},
		"simple positive slope calculation": {
			dataPoints: []DataPoint{
				{0.0, 0.0},
				{1.0, 1.0},
				{2.0, 2.0},
				{3.0, 3.0},
				{4.0, 4.0},
				{5.0, 5.0},
			},
			expected: 1.0,
		},
		"simple negative slope calculation": {
			dataPoints: []DataPoint{
				{0.0, 5.0},
				{1.0, 4.0},
				{2.0, 3.0},
				{3.0, 2.0},
				{4.0, 1.0},
				{5.0, 0.0},
			},
			expected: -1.0,
		},
		"simple zero slope calculation": {
			dataPoints: []DataPoint{
				{0.0, 1.0},
				{1.0, 1.0},
				{2.0, 1.0},
				{3.0, 1.0},
				{4.0, 1.0},
				{5.0, 1.0},
			},
			expected: 0.0,
		},
		"complex positive slope calculation": {
			dataPoints: []DataPoint{
				{0.0, 10.0},
				{1.0, 12.0},
				{2.0, 11.0},
				{3.0, 13.0},
				{4.0, 15.0},
				{5.0, 16.0},
			},
			expected: defaultRecordsSlope,
		},
		"complex negative slope calculation": {
			dataPoints: []DataPoint{
				{5.0, 10.0},
				{4.0, 12.0},
				{3.0, 11.0},
				{2.0, 13.0},
				{1.0, 15.0},
				{0.0, 16.0},
			},
			expected: -defaultRecordsSlope,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			actual, err := CalculateLinearRegressionSlope(tc.dataPoints)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
