package lender

import (
	"testing"

	"github.com/sassoftware/gopher-hole/metrics"
	"github.com/stretchr/testify/assert"
)

// MockPredictor implements Predictor interface for testing BaseModel
type MockPredictor struct {
	predictions []float64
}

func (m *MockPredictor) PredictSample(_ []float64) []float64 {
	return m.predictions
}

func TestBaseModel_ListMetrics(t *testing.T) {
	tests := map[string]struct {
		metrics  []*metrics.Metric
		expected int
	}{
		"empty metrics": {
			metrics:  nil,
			expected: 0,
		},
		"single metric": {
			metrics:  []*metrics.Metric{metrics.NewMetric("cpu", []string{"server"})},
			expected: 1,
		},
		"multiple metrics": {
			metrics: []*metrics.Metric{
				metrics.NewMetric("cpu", []string{"server"}),
				metrics.NewMetric("memory", []string{"server"}),
				metrics.NewMetric("disk", []string{"server"}),
			},
			expected: 3,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			baseModel := &BaseModel{
				Metrics: tc.metrics,
			}

			result := baseModel.ListMetrics()
			assert.Equal(t, tc.expected, len(result))
			assert.Equal(t, tc.metrics, result)
		})
	}
}

func TestBaseModel_Evaluate_EmptySamples(t *testing.T) {
	baseModel := &BaseModel{
		Predictor: &MockPredictor{predictions: []float64{0.5}},
	}

	result := baseModel.Evaluate([]TrainingSample{}, ScaleUpThresholdDefault)

	assert.Equal(t, ModelEval{}, result)
}

func TestBaseModel_Evaluate_NilPredictor(t *testing.T) {
	baseModel := &BaseModel{
		Predictor: nil,
	}

	samples := []TrainingSample{
		{Inputs: []float64{1.0}, Expected: []float64{1.0}},
	}

	result := baseModel.Evaluate(samples, ScaleUpThresholdDefault)

	assert.Equal(t, ModelEval{}, result)
}

func TestBaseModel_Evaluate_PerfectPredictions(t *testing.T) {
	// Predictor that always returns exactly what's expected
	baseModel := &BaseModel{
		Predictor: &MockPredictor{predictions: []float64{0.9}},
	}

	samples := []TrainingSample{
		{Inputs: []float64{1.0, 2.0}, Expected: []float64{0.9}},
		{Inputs: []float64{3.0, 4.0}, Expected: []float64{0.9}},
	}

	result := baseModel.Evaluate(samples, ScaleUpThresholdDefault)

	assert.Equal(t, 1.0, result.Accuracy)
	assert.Equal(t, 0.0, result.Loss)
	assert.Equal(t, 1.0, result.Precision)
	assert.Equal(t, 1.0, result.Recall)
}

func TestBaseModel_Evaluate_AllTruePositives(t *testing.T) {
	// Predictor returns high value (above threshold)
	baseModel := &BaseModel{
		Predictor: &MockPredictor{predictions: []float64{0.8}},
	}

	// All expected values are also above threshold
	samples := []TrainingSample{
		{Inputs: []float64{1.0}, Expected: []float64{0.9}},
		{Inputs: []float64{2.0}, Expected: []float64{0.7}},
		{Inputs: []float64{3.0}, Expected: []float64{0.6}},
	}

	result := baseModel.Evaluate(samples, ScaleUpThresholdDefault)

	assert.Equal(t, 1.0, result.Accuracy)
	assert.Equal(t, 1.0, result.Precision)
	assert.Equal(t, 1.0, result.Recall)
}

func TestBaseModel_Evaluate_AllTrueNegatives(t *testing.T) {
	// Predictor returns low value (below threshold)
	baseModel := &BaseModel{
		Predictor: &MockPredictor{predictions: []float64{0.2}},
	}

	// All expected values are also below threshold
	samples := []TrainingSample{
		{Inputs: []float64{1.0}, Expected: []float64{0.1}},
		{Inputs: []float64{2.0}, Expected: []float64{0.3}},
		{Inputs: []float64{3.0}, Expected: []float64{0.4}},
	}

	result := baseModel.Evaluate(samples, ScaleUpThresholdDefault)

	assert.Equal(t, 1.0, result.Accuracy)
	assert.Equal(t, 0.0, result.Precision) // No positive predictions
	assert.Equal(t, 0.0, result.Recall)    // No actual positives
}

func TestBaseModel_Evaluate_MixedResults(t *testing.T) {
	// Predictor returns 0.6 (above 0.5 threshold)
	baseModel := &BaseModel{
		Predictor: &MockPredictor{predictions: []float64{0.6}},
	}

	// Mix of expected values above and below threshold
	samples := []TrainingSample{
		{Inputs: []float64{1.0}, Expected: []float64{0.8}}, // True positive
		{Inputs: []float64{2.0}, Expected: []float64{0.7}}, // True positive
		{Inputs: []float64{3.0}, Expected: []float64{0.2}}, // False positive
		{Inputs: []float64{4.0}, Expected: []float64{0.1}}, // False positive
	}

	result := baseModel.Evaluate(samples, ScaleUpThresholdDefault)

	// 2 TP + 0 TN = 2 correct out of 4
	assert.Equal(t, 0.5, result.Accuracy)
	// 2 TP / (2 TP + 2 FP) = 0.5
	assert.Equal(t, 0.5, result.Precision)
	// 2 TP / (2 TP + 0 FN) = 1.0
	assert.Equal(t, 1.0, result.Recall)
}

func TestBaseModel_Evaluate_LossCalculation(t *testing.T) {
	// Predictor returns 0.5
	baseModel := &BaseModel{
		Predictor: &MockPredictor{predictions: []float64{0.5}},
	}

	samples := []TrainingSample{
		{Inputs: []float64{1.0}, Expected: []float64{1.0}}, // Error = 0.5, MSE = 0.25
		{Inputs: []float64{2.0}, Expected: []float64{0.0}}, // Error = 0.5, MSE = 0.25
	}

	result := baseModel.Evaluate(samples, ScaleUpThresholdDefault)

	// Average MSE = (0.25 + 0.25) / 2 = 0.25
	assert.Equal(t, 0.25, result.Loss)
}

func TestTrainingSample_Structure(t *testing.T) {
	sample := TrainingSample{
		Inputs:   []float64{1.0, 2.0, 3.0},
		Expected: []float64{0.5},
	}

	assert.Equal(t, 3, len(sample.Inputs))
	assert.Equal(t, 1, len(sample.Expected))
	assert.Equal(t, 1.0, sample.Inputs[0])
	assert.Equal(t, 0.5, sample.Expected[0])
}

func TestModelEval_Structure(t *testing.T) {
	eval := ModelEval{
		Accuracy:  0.95,
		Loss:      0.05,
		Precision: 0.92,
		Recall:    0.88,
	}

	assert.Equal(t, 0.95, eval.Accuracy)
	assert.Equal(t, 0.05, eval.Loss)
	assert.Equal(t, 0.92, eval.Precision)
	assert.Equal(t, 0.88, eval.Recall)
}

func TestApplyNormalization(t *testing.T) {
	tests := map[string]struct {
		fn       *FeatureNormalization
		input    float64
		expected float64
	}{
		"minmax mid-range": {
			fn:       &FeatureNormalization{Type: NormalizationMinMax, Min: 0, Max: 100},
			input:    50,
			expected: 0.5,
		},
		"minmax at min": {
			fn:       &FeatureNormalization{Type: NormalizationMinMax, Min: 0, Max: 100},
			input:    0,
			expected: 0.0,
		},
		"minmax at max": {
			fn:       &FeatureNormalization{Type: NormalizationMinMax, Min: 0, Max: 100},
			input:    100,
			expected: 1.0,
		},
		"minmax min equals max returns zero": {
			fn:       &FeatureNormalization{Type: NormalizationMinMax, Min: 50, Max: 50},
			input:    50,
			expected: 0.0,
		},
		"minmax_symmetric mid-range": {
			fn:       &FeatureNormalization{Type: NormalizationMinMaxSymmetric, Min: 0, Max: 100},
			input:    50,
			expected: 0.0,
		},
		"minmax_symmetric at min": {
			fn:       &FeatureNormalization{Type: NormalizationMinMaxSymmetric, Min: 0, Max: 100},
			input:    0,
			expected: -1.0,
		},
		"minmax_symmetric at max": {
			fn:       &FeatureNormalization{Type: NormalizationMinMaxSymmetric, Min: 0, Max: 100},
			input:    100,
			expected: 1.0,
		},
		"minmax_symmetric min equals max returns zero": {
			fn:       &FeatureNormalization{Type: NormalizationMinMaxSymmetric, Min: 50, Max: 50},
			input:    50,
			expected: 0.0,
		},
		"zscore mid-range": {
			fn:       &FeatureNormalization{Type: NormalizationZScore, Mean: 50, StdDev: 10},
			input:    60,
			expected: 1.0,
		},
		"zscore at mean": {
			fn:       &FeatureNormalization{Type: NormalizationZScore, Mean: 50, StdDev: 10},
			input:    50,
			expected: 0.0,
		},
		"zscore stddev zero returns zero": {
			fn:       &FeatureNormalization{Type: NormalizationZScore, Mean: 50, StdDev: 0},
			input:    60,
			expected: 0.0,
		},
		"unknown type passes through unchanged": {
			fn:       &FeatureNormalization{Type: "unknown"},
			input:    42.5,
			expected: 42.5,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := ApplyNormalization(tc.input, tc.fn)
			assert.InDelta(t, tc.expected, result, 0.0001)
		})
	}
}
