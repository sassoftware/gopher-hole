package lender

import (
	"math"
	"testing"

	"github.com/sassoftware/gopher-hole/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewShallowNeuralNetwork(t *testing.T) {
	metricsList := []*metrics.Metric{
		metrics.NewMetric("cpu", []string{"server"}),
		metrics.NewMetric("memory", []string{"server"}),
	}

	nn := NewShallowNeuralNetwork(2, 4, metricsList)

	require.NotNil(t, nn)
	inputs, hidden, outputs := nn.GetArchitecture()
	assert.Equal(t, 2, inputs)
	assert.Equal(t, 4, hidden)
	assert.Equal(t, 1, outputs) // Default single output
	assert.Equal(t, metricsList, nn.ListMetrics())
}

func TestNewShallowNeuralNetworkWithOutputs(t *testing.T) {
	metricsList := []*metrics.Metric{
		metrics.NewMetric("cpu", []string{"server"}),
	}

	nn := NewShallowNeuralNetworkWithOutputs(3, 5, 2, metricsList)

	require.NotNil(t, nn)
	inputs, hidden, outputs := nn.GetArchitecture()
	assert.Equal(t, 3, inputs)
	assert.Equal(t, 5, hidden)
	assert.Equal(t, 2, outputs)
}

func TestShallowNeuralNetwork_WeightInitialization(t *testing.T) {
	nn := NewShallowNeuralNetwork(3, 4, nil)

	// Check weight dimensions
	assert.Equal(t, 3, len(nn.weightsInputHidden))
	for i := 0; i < 3; i++ {
		assert.Equal(t, 4, len(nn.weightsInputHidden[i]))
	}

	assert.Equal(t, 4, len(nn.biasesHidden))
	assert.Equal(t, 4, len(nn.weightsHiddenOutput))
	for i := 0; i < 4; i++ {
		assert.Equal(t, 1, len(nn.weightsHiddenOutput[i]))
	}
	assert.Equal(t, 1, len(nn.biasesOutput))

	// Verify weights are initialized (not all zeros)
	hasNonZero := false
	for i := 0; i < 3; i++ {
		for j := 0; j < 4; j++ {
			if nn.weightsInputHidden[i][j] != 0 {
				hasNonZero = true
				break
			}
		}
	}
	assert.True(t, hasNonZero, "Weights should be initialized to non-zero values")
}

func TestShallowNeuralNetwork_PredictSample(t *testing.T) {
	nn := NewShallowNeuralNetwork(2, 3, nil)

	outputs := nn.PredictSample([]float64{0.5, 0.5})

	require.Equal(t, 1, len(outputs))
	// Output should be between 0 and 1 (sigmoid activation)
	assert.True(t, outputs[0] >= 0 && outputs[0] <= 1)
}

func TestShallowNeuralNetwork_Predict(t *testing.T) {
	metric1 := metrics.NewMetric("cpu", []string{"server"})
	metric2 := metrics.NewMetric("memory", []string{"server"})

	metric1.InsertRecord(0.7)
	metric2.InsertRecord(0.3)

	metricsList := []*metrics.Metric{metric1, metric2}
	nn := NewShallowNeuralNetwork(2, 3, metricsList)

	prediction, err := nn.Predict(metricsList)
	require.NoError(t, err)

	// Prediction should be between 0 and 1
	assert.True(t, prediction >= 0 && prediction <= 1)
}

func TestShallowNeuralNetwork_Predict_PaddingInputs(t *testing.T) {
	// Create network expecting 3 inputs
	nn := NewShallowNeuralNetwork(3, 2, nil)

	// Provide only 1 metric
	metric := metrics.NewMetric("cpu", []string{"server"})
	metric.InsertRecord(0.5)

	prediction, err := nn.Predict([]*metrics.Metric{metric})
	require.NoError(t, err)

	// Should still produce valid output (padding with zeros)
	assert.True(t, prediction >= 0 && prediction <= 1)
}

func TestShallowNeuralNetwork_Predict_TruncatingInputs(t *testing.T) {
	// Create network expecting 2 inputs
	nn := NewShallowNeuralNetwork(2, 2, nil)

	// Provide 4 metrics
	metricsList := make([]*metrics.Metric, 4)
	for i := 0; i < 4; i++ {
		metricsList[i] = metrics.NewMetric("metric", []string{"server"})
		metricsList[i].InsertRecord(float64(i) * 0.25)
	}

	prediction, err := nn.Predict(metricsList)
	require.NoError(t, err)

	// Should still produce valid output (truncating extras)
	assert.True(t, prediction >= 0 && prediction <= 1)
}

func TestShallowNeuralNetwork_Train_XOR(t *testing.T) {
	// XOR is a classic test for neural networks
	// It's not linearly separable, so it tests the hidden layer
	nn := NewShallowNeuralNetwork(2, 4, nil)

	samples := []TrainingSample{
		{Inputs: []float64{0, 0}, Expected: []float64{0}},
		{Inputs: []float64{0, 1}, Expected: []float64{1}},
		{Inputs: []float64{1, 0}, Expected: []float64{1}},
		{Inputs: []float64{1, 1}, Expected: []float64{0}},
	}

	// Train for many epochs
	err := nn.Train(samples, 5000, 1.0)
	require.NoError(t, err)

	// Test predictions
	result00 := nn.PredictSample([]float64{0, 0})[0]
	result01 := nn.PredictSample([]float64{0, 1})[0]
	result10 := nn.PredictSample([]float64{1, 0})[0]
	result11 := nn.PredictSample([]float64{1, 1})[0]

	// Allow some tolerance
	tolerance := 0.3
	assert.True(t, result00 < 0.5+tolerance, "XOR(0,0) should be close to 0, got %f", result00)
	assert.True(t, result01 > 0.5-tolerance, "XOR(0,1) should be close to 1, got %f", result01)
	assert.True(t, result10 > 0.5-tolerance, "XOR(1,0) should be close to 1, got %f", result10)
	assert.True(t, result11 < 0.5+tolerance, "XOR(1,1) should be close to 0, got %f", result11)
}

func TestShallowNeuralNetwork_Train_AND(t *testing.T) { //nolint:dupl
	nn := NewShallowNeuralNetwork(2, 2, nil)

	samples := []TrainingSample{
		{Inputs: []float64{0, 0}, Expected: []float64{0}},
		{Inputs: []float64{0, 1}, Expected: []float64{0}},
		{Inputs: []float64{1, 0}, Expected: []float64{0}},
		{Inputs: []float64{1, 1}, Expected: []float64{1}},
	}

	err := nn.Train(samples, 1000, 2.0)
	require.NoError(t, err)

	result00 := nn.PredictSample([]float64{0, 0})[0]
	result01 := nn.PredictSample([]float64{0, 1})[0]
	result10 := nn.PredictSample([]float64{1, 0})[0]
	result11 := nn.PredictSample([]float64{1, 1})[0]

	assert.True(t, result00 < 0.5, "AND(0,0) should be < 0.5, got %f", result00)
	assert.True(t, result01 < 0.5, "AND(0,1) should be < 0.5, got %f", result01)
	assert.True(t, result10 < 0.5, "AND(1,0) should be < 0.5, got %f", result10)
	assert.True(t, result11 > 0.5, "AND(1,1) should be > 0.5, got %f", result11)
}

func TestShallowNeuralNetwork_Train_OR(t *testing.T) { //nolint:dupl
	nn := NewShallowNeuralNetwork(2, 2, nil)

	samples := []TrainingSample{
		{Inputs: []float64{0, 0}, Expected: []float64{0}},
		{Inputs: []float64{0, 1}, Expected: []float64{1}},
		{Inputs: []float64{1, 0}, Expected: []float64{1}},
		{Inputs: []float64{1, 1}, Expected: []float64{1}},
	}

	err := nn.Train(samples, 1000, 2.0)
	require.NoError(t, err)

	result00 := nn.PredictSample([]float64{0, 0})[0]
	result01 := nn.PredictSample([]float64{0, 1})[0]
	result10 := nn.PredictSample([]float64{1, 0})[0]
	result11 := nn.PredictSample([]float64{1, 1})[0]

	assert.True(t, result00 < 0.5, "OR(0,0) should be < 0.5, got %f", result00)
	assert.True(t, result01 > 0.5, "OR(0,1) should be > 0.5, got %f", result01)
	assert.True(t, result10 > 0.5, "OR(1,0) should be > 0.5, got %f", result10)
	assert.True(t, result11 > 0.5, "OR(1,1) should be > 0.5, got %f", result11)
}

func TestShallowNeuralNetwork_ExportImport(t *testing.T) {
	// Create and train a network
	nn1 := NewShallowNeuralNetwork(2, 3, nil)
	samples := []TrainingSample{
		{Inputs: []float64{0, 0}, Expected: []float64{0}},
		{Inputs: []float64{1, 1}, Expected: []float64{1}},
	}
	err := nn1.Train(samples, 100, 1.0)
	require.NoError(t, err)

	// Export the trained network
	exported := nn1.Export()
	require.NotNil(t, exported)
	require.True(t, len(exported) > 0)

	// Create a new network and import
	nn2 := NewShallowNeuralNetwork(2, 3, nil)
	err = nn2.Import(exported)
	require.NoError(t, err)

	// Verify architecture matches
	inputs1, hidden1, outputs1 := nn1.GetArchitecture()
	inputs2, hidden2, outputs2 := nn2.GetArchitecture()
	assert.Equal(t, inputs1, inputs2)
	assert.Equal(t, hidden1, hidden2)
	assert.Equal(t, outputs1, outputs2)

	// Verify predictions match
	testInputs := []float64{0.5, 0.5}
	pred1 := nn1.PredictSample(testInputs)
	pred2 := nn2.PredictSample(testInputs)
	assert.InDelta(t, pred1[0], pred2[0], 1e-10)
}

func TestShallowNeuralNetwork_Import_InvalidJSON(t *testing.T) {
	nn := NewShallowNeuralNetwork(2, 3, nil)

	err := nn.Import([]byte("invalid json"))

	assert.Error(t, err)
}

func TestShallowNeuralNetwork_Import_InvalidModelType(t *testing.T) {
	nn := NewShallowNeuralNetwork(2, 3, nil)

	invalidData := `{
		"modelType": "DifferentModel",
		"numInputs": 2,
		"numHidden": 3,
		"numOutputs": 1,
		"weightsInputHidden": [[0.1, 0.2, 0.3], [0.4, 0.5, 0.6]],
		"biasesHidden": [0.1, 0.2, 0.3],
		"weightsHiddenOutput": [[0.1], [0.2], [0.3]],
		"biasesOutput": [0.1]
	}`

	err := nn.Import([]byte(invalidData))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incompatible model type")
}

func TestShallowNeuralNetwork_Import_InvalidDimensions(t *testing.T) {
	tests := map[string]struct {
		jsonData      string
		expectedError string
	}{
		"zero inputs": {
			jsonData: `{
				"numInputs": 0,
				"numHidden": 3,
				"numOutputs": 1,
				"weightsInputHidden": [],
				"biasesHidden": [0.1, 0.2, 0.3],
				"weightsHiddenOutput": [[0.1], [0.2], [0.3]],
				"biasesOutput": [0.1]
			}`,
			expectedError: "invalid architecture dimensions",
		},
		"weights mismatch": {
			jsonData: `{
				"numInputs": 2,
				"numHidden": 3,
				"numOutputs": 1,
				"weightsInputHidden": [[0.1, 0.2, 0.3]],
				"biasesHidden": [0.1, 0.2, 0.3],
				"weightsHiddenOutput": [[0.1], [0.2], [0.3]],
				"biasesOutput": [0.1]
			}`,
			expectedError: "weightsInputHidden dimension mismatch",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			nn := NewShallowNeuralNetwork(2, 3, nil)
			err := nn.Import([]byte(tc.jsonData))
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

func TestShallowNeuralNetwork_Import_UnsupportedActivation(t *testing.T) {
	nn := NewShallowNeuralNetwork(2, 3, nil)

	invalidData := `{
		"numInputs": 2,
		"numHidden": 3,
		"numOutputs": 1,
		"weightsInputHidden": [[0.1, 0.2, 0.3], [0.4, 0.5, 0.6]],
		"biasesHidden": [0.1, 0.2, 0.3],
		"weightsHiddenOutput": [[0.1], [0.2], [0.3]],
		"biasesOutput": [0.1],
		"hiddenActivation": "relu",
		"outputActivation": "sigmoid"
	}`

	err := nn.Import([]byte(invalidData))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported hidden activation")
}

func TestShallowNeuralNetwork_JavaCompatibility(t *testing.T) {
	// Test that we can import a Java-formatted model
	javaFormattedJSON := `{
		"modelType": "ShallowNeuralNetwork",
		"version": "1.0",
		"createdAt": "2026-01-19T00:00:00Z",
		"description": "Test model from Java",
		"numInputs": 2,
		"numHidden": 3,
		"numOutputs": 1,
		"weightsInputHidden": [[0.1, 0.2, 0.3], [0.4, 0.5, 0.6]],
		"biasesHidden": [0.1, 0.2, 0.3],
		"weightsHiddenOutput": [[0.7], [0.8], [0.9]],
		"biasesOutput": [0.5],
		"hiddenActivation": "sigmoid",
		"outputActivation": "sigmoid"
	}`

	nn := NewShallowNeuralNetwork(2, 3, nil)
	err := nn.Import([]byte(javaFormattedJSON))
	require.NoError(t, err)

	// Verify architecture was loaded correctly
	inputs, hidden, outputs := nn.GetArchitecture()
	assert.Equal(t, 2, inputs)
	assert.Equal(t, 3, hidden)
	assert.Equal(t, 1, outputs)

	// Verify weights were loaded
	assert.Equal(t, 0.1, nn.weightsInputHidden[0][0])
	assert.Equal(t, 0.5, nn.biasesOutput[0])
}

func TestShallowNeuralNetwork_Export_Format(t *testing.T) {
	nn := NewShallowNeuralNetwork(2, 3, nil)

	exported := nn.Export()

	// Should be valid JSON containing expected fields
	assert.Contains(t, string(exported), "modelType")
	assert.Contains(t, string(exported), "version")
	assert.Contains(t, string(exported), "numInputs")
	assert.Contains(t, string(exported), "numHidden")
	assert.Contains(t, string(exported), "numOutputs")
	assert.Contains(t, string(exported), "weightsInputHidden")
	assert.Contains(t, string(exported), "biasesHidden")
	assert.Contains(t, string(exported), "weightsHiddenOutput")
	assert.Contains(t, string(exported), "biasesOutput")
	assert.Contains(t, string(exported), "hiddenActivation")
	assert.Contains(t, string(exported), "outputActivation")
	assert.Contains(t, string(exported), "ShallowNeuralNetwork")
	assert.Contains(t, string(exported), "sigmoid")
}

func TestShallowNeuralNetwork_Evaluate(t *testing.T) {
	nn := NewShallowNeuralNetwork(2, 4, nil)

	// Train on simple pattern
	trainingSamples := []TrainingSample{
		{Inputs: []float64{0.9, 0.9}, Expected: []float64{1.0}},
		{Inputs: []float64{0.1, 0.1}, Expected: []float64{0.0}},
	}
	err := nn.Train(trainingSamples, 1000, 2.0)
	require.NoError(t, err)

	// Evaluate on test samples
	testSamples := []TrainingSample{
		{Inputs: []float64{0.8, 0.8}, Expected: []float64{1.0}},
		{Inputs: []float64{0.2, 0.2}, Expected: []float64{0.0}},
	}

	eval := nn.Evaluate(testSamples, ScaleUpThresholdDefault)

	// Should have reasonable metrics after training
	assert.True(t, eval.Accuracy >= 0 && eval.Accuracy <= 1)
	assert.True(t, eval.Loss >= 0)
	assert.True(t, eval.Precision >= 0 && eval.Precision <= 1)
	assert.True(t, eval.Recall >= 0 && eval.Recall <= 1)
}

func TestShallowNeuralNetwork_MultipleOutputs(t *testing.T) {
	nn := NewShallowNeuralNetworkWithOutputs(2, 3, 2, nil)

	outputs := nn.PredictSample([]float64{0.5, 0.5})

	require.Equal(t, 2, len(outputs))
	assert.True(t, outputs[0] >= 0 && outputs[0] <= 1)
	assert.True(t, outputs[1] >= 0 && outputs[1] <= 1)
}

func TestShallowNeuralNetwork_TrainMultipleOutputs(t *testing.T) {
	nn := NewShallowNeuralNetworkWithOutputs(2, 4, 2, nil)

	// Train to output both AND and OR
	samples := []TrainingSample{
		{Inputs: []float64{0, 0}, Expected: []float64{0, 0}}, // AND=0, OR=0
		{Inputs: []float64{0, 1}, Expected: []float64{0, 1}}, // AND=0, OR=1
		{Inputs: []float64{1, 0}, Expected: []float64{0, 1}}, // AND=0, OR=1
		{Inputs: []float64{1, 1}, Expected: []float64{1, 1}}, // AND=1, OR=1
	}

	err := nn.Train(samples, 2000, 1.0)
	require.NoError(t, err)

	// Check predictions
	result := nn.PredictSample([]float64{1, 1})
	assert.True(t, result[0] > 0.5, "AND(1,1) should be > 0.5, got %f", result[0])
	assert.True(t, result[1] > 0.5, "OR(1,1) should be > 0.5, got %f", result[1])

	result = nn.PredictSample([]float64{0, 0})
	assert.True(t, result[0] < 0.5, "AND(0,0) should be < 0.5, got %f", result[0])
	assert.True(t, result[1] < 0.5, "OR(0,0) should be < 0.5, got %f", result[1])
}

func TestSigmoid(t *testing.T) {
	tests := map[string]struct {
		input    float64
		expected float64
	}{
		"zero": {
			input:    0,
			expected: 0.5,
		},
		"large positive": {
			input:    10,
			expected: 0.9999546,
		},
		"large negative": {
			input:    -10,
			expected: 0.0000454,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := sigmoid(tc.input)
			assert.InDelta(t, tc.expected, result, 0.0001)
		})
	}
}

func TestSigmoidDerivative(t *testing.T) {
	tests := map[string]struct {
		sigmoidOutput float64
		expected      float64
	}{
		"at 0.5": {
			sigmoidOutput: 0.5,
			expected:      0.25, // 0.5 * (1 - 0.5) = 0.25
		},
		"at 0": {
			sigmoidOutput: 0,
			expected:      0, // 0 * 1 = 0
		},
		"at 1": {
			sigmoidOutput: 1,
			expected:      0, // 1 * 0 = 0
		},
		"at 0.7": {
			sigmoidOutput: 0.7,
			expected:      0.21, // 0.7 * 0.3 = 0.21
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := sigmoidDerivative(tc.sigmoidOutput)
			assert.InDelta(t, tc.expected, result, 0.0001)
		})
	}
}

func TestShallowNeuralNetwork_ForwardDeterministic(t *testing.T) {
	nn := NewShallowNeuralNetwork(2, 3, nil)

	inputs := []float64{0.5, 0.7}

	// Forward pass should be deterministic
	_, output1 := nn.forward(inputs)
	_, output2 := nn.forward(inputs)

	assert.Equal(t, output1, output2)
}

func TestShallowNeuralNetwork_EmptyTrainingSamples(t *testing.T) {
	nn := NewShallowNeuralNetwork(2, 3, nil)

	// Get initial prediction
	initialPred := nn.PredictSample([]float64{0.5, 0.5})[0]

	// Train with empty samples - should return error
	err := nn.Train([]TrainingSample{}, 100, 1.0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "training samples cannot be empty")

	// Prediction should be unchanged since training failed
	finalPred := nn.PredictSample([]float64{0.5, 0.5})[0]
	assert.Equal(t, initialPred, finalPred)
}

func TestShallowNeuralNetwork_LearningRateEffect(t *testing.T) {
	samples := []TrainingSample{
		{Inputs: []float64{1, 1}, Expected: []float64{1}},
		{Inputs: []float64{0, 0}, Expected: []float64{0}},
	}

	// Train with low learning rate
	nn1 := NewShallowNeuralNetwork(2, 3, nil)
	err := nn1.Train(samples, 10, 0.01)
	require.NoError(t, err)
	pred1 := nn1.PredictSample([]float64{1, 1})[0]

	// Train with high learning rate
	nn2 := NewShallowNeuralNetwork(2, 3, nil)
	err = nn2.Train(samples, 10, 1.0)
	require.NoError(t, err)
	pred2 := nn2.PredictSample([]float64{1, 1})[0]

	// Higher learning rate should move predictions more quickly
	// (though not always in the right direction)
	assert.NotEqual(t, pred1, pred2, "Different learning rates should produce different results")
}

func TestShallowNeuralNetwork_Train_ValidationErrors(t *testing.T) {
	nn := NewShallowNeuralNetwork(2, 3, nil)

	// Test empty samples
	err := nn.Train([]TrainingSample{}, 100, 1.0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "training samples cannot be empty")

	// Test negative epochs
	samples := []TrainingSample{
		{Inputs: []float64{0.5, 0.5}, Expected: []float64{1.0}},
	}
	err = nn.Train(samples, -10, 1.0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "epochs must be positive")

	// Test zero epochs
	err = nn.Train(samples, 0, 1.0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "epochs must be positive")

	// Test negative learning rate
	err = nn.Train(samples, 100, -0.5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "learning rate must be positive")

	// Test zero learning rate
	err = nn.Train(samples, 100, 0.0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "learning rate must be positive")

	// Test sample with inccorect number of inputs
	samplesWrongInputs := []TrainingSample{
		{Inputs: []float64{0.5}, Expected: []float64{1.0}},
	}
	err = nn.Train(samplesWrongInputs, 100, 1.0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has 1 inputs, expected 2")

	// Test sample with wrong number of outputs
	samplesWrongOutputs := []TrainingSample{
		{Inputs: []float64{0.5, 0.5}, Expected: []float64{1.0, 0.5}},
	}
	err = nn.Train(samplesWrongOutputs, 100, 1.0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has 2 outputs, expected 1")
}

func TestShallowNeuralNetwork_ImplementsInterfaces(_ *testing.T) {
	nn := NewShallowNeuralNetwork(2, 3, nil)

	// Verify ShallowNeuralNetwork implements AIModel
	var _ AIModel = nn

	// Verify ShallowNeuralNetwork implements Trainable
	var _ Trainable = nn

	// Verify ShallowNeuralNetwork implements Predictor
	var _ Predictor = nn
}

func TestShallowNeuralNetwork_NilMetricsInPredict(t *testing.T) {
	nn := NewShallowNeuralNetwork(2, 3, nil)

	// Include nil metrics in the list
	metricsList := []*metrics.Metric{nil, nil}

	// Should handle gracefully without panic
	prediction, err := nn.Predict(metricsList)
	require.NoError(t, err)

	// Should still produce valid output (with zeros for nil metrics)
	assert.True(t, prediction >= 0 && prediction <= 1)
}

func TestShallowNeuralNetwork_NilMetricsList(t *testing.T) {
	nn := NewShallowNeuralNetwork(2, 3, nil)

	// Pass nil for the entire metricsList
	prediction, err := nn.Predict(nil)
	require.Error(t, err)
	assert.Equal(t, 0.0, prediction)
	assert.Contains(t, err.Error(), "metricsList cannot be nil")
}

func TestShallowNeuralNetwork_OutputBounds(t *testing.T) {
	nn := NewShallowNeuralNetwork(2, 3, nil)

	// Test with extreme input values
	testCases := [][]float64{
		{0, 0},
		{1, 1},
		{-1000, -1000},
		{1000, 1000},
		{math.MaxFloat64, math.MaxFloat64},
		{-math.MaxFloat64, -math.MaxFloat64},
	}

	for _, inputs := range testCases {
		outputs := nn.PredictSample(inputs)
		for i, output := range outputs {
			// Check for NaN or Inf values which indicate numerical issues
			assert.False(t, math.IsNaN(output),
				"Output %d is NaN for inputs %v", i, inputs)
			assert.False(t, math.IsInf(output, 0),
				"Output %d is Inf for inputs %v", i, inputs)
			// Check bounds only for valid numbers
			if !math.IsNaN(output) && !math.IsInf(output, 0) {
				assert.True(t, output >= 0 && output <= 1,
					"Output %d should be bounded [0,1] for inputs %v, got %f", i, inputs, output)
			}
		}
	}
}

func TestShallowNeuralNetwork_SetFeatureNormalization_MinMax(t *testing.T) {
	nn := NewShallowNeuralNetwork(3, 2, nil)

	// Set min-max normalization for feature 1
	err := nn.SetFeatureNormalization(1, NormalizationMinMax, 100, 200, 0, 0)
	require.NoError(t, err)

	assert.Len(t, nn.featureNormalizations, 1)
	assert.Equal(t, 1, nn.featureNormalizations[0].FeatureIndex)
	assert.Equal(t, NormalizationMinMax, nn.featureNormalizations[0].Type)
	assert.Equal(t, 100.0, nn.featureNormalizations[0].Min)
	assert.Equal(t, 200.0, nn.featureNormalizations[0].Max)
}

func TestShallowNeuralNetwork_SetFeatureNormalization_ZScore(t *testing.T) {
	nn := NewShallowNeuralNetwork(3, 2, nil)

	// Set z-score normalization for feature 0
	err := nn.SetFeatureNormalization(0, NormalizationZScore, 0, 0, 50, 10)
	require.NoError(t, err)

	assert.Len(t, nn.featureNormalizations, 1)
	assert.Equal(t, 0, nn.featureNormalizations[0].FeatureIndex)
	assert.Equal(t, NormalizationZScore, nn.featureNormalizations[0].Type)
	assert.Equal(t, 50.0, nn.featureNormalizations[0].Mean)
	assert.Equal(t, 10.0, nn.featureNormalizations[0].StdDev)
}

func TestShallowNeuralNetwork_SetFeatureNormalization_InvalidIndex(t *testing.T) {
	nn := NewShallowNeuralNetwork(3, 2, nil)

	// Test negative index
	err := nn.SetFeatureNormalization(-1, NormalizationMinMax, 0, 100, 0, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")

	// Test index too large
	err = nn.SetFeatureNormalization(3, NormalizationMinMax, 0, 100, 0, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestShallowNeuralNetwork_SetFeatureNormalization_InvalidMinMax(t *testing.T) {
	nn := NewShallowNeuralNetwork(3, 2, nil)

	// Test min >= max
	err := nn.SetFeatureNormalization(0, NormalizationMinMax, 100, 100, 0, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "min < max")

	err = nn.SetFeatureNormalization(0, NormalizationMinMax, 100, 50, 0, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "min < max")
}

func TestShallowNeuralNetwork_SetFeatureNormalization_InvalidStdDev(t *testing.T) {
	nn := NewShallowNeuralNetwork(3, 2, nil)

	// Test stdDev <= 0
	err := nn.SetFeatureNormalization(0, NormalizationZScore, 0, 0, 50, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stdDev > 0")

	err = nn.SetFeatureNormalization(0, NormalizationZScore, 0, 0, 50, -5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stdDev > 0")
}

func TestShallowNeuralNetwork_SetFeatureNormalization_Replace(t *testing.T) {
	nn := NewShallowNeuralNetwork(3, 2, nil)

	// Set initial normalization
	err := nn.SetFeatureNormalization(1, NormalizationMinMax, 0, 100, 0, 0)
	require.NoError(t, err)
	assert.Len(t, nn.featureNormalizations, 1)

	// Replace with different normalization
	err = nn.SetFeatureNormalization(1, NormalizationZScore, 0, 0, 50, 10)
	require.NoError(t, err)
	assert.Len(t, nn.featureNormalizations, 1)
	assert.Equal(t, NormalizationZScore, nn.featureNormalizations[0].Type)
	assert.Equal(t, 50.0, nn.featureNormalizations[0].Mean)
}

func TestShallowNeuralNetwork_ApplyNormalizations_MinMax(t *testing.T) {
	nn := NewShallowNeuralNetwork(3, 2, nil)

	// Configure min-max normalization: [100, 200] -> [0, 1]
	err := nn.SetFeatureNormalization(1, NormalizationMinMax, 100, 200, 0, 0)
	require.NoError(t, err)

	// Test normalization
	inputs := []float64{0.5, 150, 0.8}
	normalized := nn.applyNormalizations(inputs)

	// Feature 0 and 2 should be unchanged
	assert.Equal(t, 0.5, normalized[0])
	assert.Equal(t, 0.8, normalized[2])

	// Feature 1 should be normalized: (150 - 100) / (200 - 100) = 0.5
	assert.InDelta(t, 0.5, normalized[1], 0.001)
}

func TestShallowNeuralNetwork_ApplyNormalizations_MinMaxIsZero(t *testing.T) {
	nn := NewShallowNeuralNetwork(3, 2, nil)

	// Configure min-max normalization: [100, 200] -> [0, 1]
	// SetFeatureNormalization already checks for min >= max, but to test
	// the behavior of applyNormalizations when min == max, we set it here
	// and modify later
	err := nn.SetFeatureNormalization(1, NormalizationMinMax, 100, 200, 0, 0)
	require.NoError(t, err)

	nn.featureNormalizations[0].Min = 100
	nn.featureNormalizations[0].Max = 100

	// Test normalization
	inputs := []float64{0.5, 150, 0.8}
	normalized := nn.applyNormalizations(inputs)

	// Feature 0 and 2 should be unchanged
	assert.Equal(t, 0.5, normalized[0])
	assert.Equal(t, 0.8, normalized[2])

	// Feature 1 should be zero because min == max
	assert.Equal(t, 0.0, normalized[1])
}

func TestShallowNeuralNetwork_ApplyNormalizations_ZScoreStdDevIsZero(t *testing.T) {
	nn := NewShallowNeuralNetwork(3, 2, nil)

	// Configure z-score normalization: mean=50, stdDev=10
	// SetFeatureNormalization already checks for min >= max, but to test
	// the behavior of applyNormalizations when stdDev == 0, we set it here
	// and modify later
	err := nn.SetFeatureNormalization(0, NormalizationZScore, 0, 0, 50, 10)
	require.NoError(t, err)
	nn.featureNormalizations[0].StdDev = 0

	// Test normalization
	inputs := []float64{60, 0.5, 0.8}
	normalized := nn.applyNormalizations(inputs)

	// Feature 1 and 2 should be unchanged
	assert.Equal(t, 0.5, normalized[1])
	assert.Equal(t, 0.8, normalized[2])

	// Feature 0 should 0.0 because StdDev is zero
	assert.Equal(t, 0.0, normalized[0])
}
func TestShallowNeuralNetwork_ApplyNormalizations_ZScore(t *testing.T) {
	nn := NewShallowNeuralNetwork(3, 2, nil)

	// Configure z-score normalization: mean=50, stdDev=10
	err := nn.SetFeatureNormalization(0, NormalizationZScore, 0, 0, 50, 10)
	require.NoError(t, err)

	// Test normalization
	inputs := []float64{60, 0.5, 0.8}
	normalized := nn.applyNormalizations(inputs)

	// Feature 1 and 2 should be unchanged
	assert.Equal(t, 0.5, normalized[1])
	assert.Equal(t, 0.8, normalized[2])

	// Feature 0 should be normalized: (60 - 50) / 10 = 1.0
	assert.InDelta(t, 1.0, normalized[0], 0.001)
}

func TestShallowNeuralNetwork_ApplyNormalizations_Multiple(t *testing.T) {
	nn := NewShallowNeuralNetwork(4, 2, nil)

	// Configure multiple normalizations
	err := nn.SetFeatureNormalization(0, NormalizationMinMax, 0, 100, 0, 0)
	require.NoError(t, err)
	err = nn.SetFeatureNormalization(2, NormalizationZScore, 0, 0, 50, 10)
	require.NoError(t, err)

	// Test normalization
	inputs := []float64{50, 0.5, 60, 0.8}
	normalized := nn.applyNormalizations(inputs)

	// Feature 0: (50 - 0) / (100 - 0) = 0.5
	assert.InDelta(t, 0.5, normalized[0], 0.001)
	// Feature 1: unchanged
	assert.Equal(t, 0.5, normalized[1])
	// Feature 2: (60 - 50) / 10 = 1.0
	assert.InDelta(t, 1.0, normalized[2], 0.001)
	// Feature 3: unchanged
	assert.Equal(t, 0.8, normalized[3])
}

func TestShallowNeuralNetwork_ApplyNormalizations_NoNormalizations(t *testing.T) {
	nn := NewShallowNeuralNetwork(3, 2, nil)

	// No normalizations configured
	inputs := []float64{1.5, 2.5, 3.5}
	normalized := nn.applyNormalizations(inputs)

	// All features should be unchanged
	assert.Equal(t, inputs, normalized)
}

func TestShallowNeuralNetwork_applyNormalization(t *testing.T) {
	nn := NewShallowNeuralNetwork(2, 3, nil)

	// Test the core functionality: that normalization is correctly applied during prediction
	// We don't need to train the network for this - just test the normalization mechanism

	// Configure normalization for feature index 1
	err := nn.SetFeatureNormalization(1, NormalizationMinMax, 100, 200, 0, 0)
	require.NoError(t, err)

	// Test that the applyNormalizations method works correctly
	rawInputs := []float64{0.5, 150}          // 150 should normalize to (150-100)/(200-100) = 0.5
	expectedNormalized := []float64{0.5, 0.5} // Both features should be 0.5 after normalization

	normalized := nn.applyNormalizations(rawInputs)
	assert.Equal(t, expectedNormalized, normalized, "Normalization should transform raw input to expected values")

	// Test edge cases of the normalization
	edgeInputs := []float64{0.5, 100} // 100 should normalize to (100-100)/(200-100) = 0.0
	expectedEdge := []float64{0.5, 0.0}
	normalizedEdge := nn.applyNormalizations(edgeInputs)
	assert.Equal(t, expectedEdge, normalizedEdge, "Normalization should handle min value correctly")

	edgeInputs2 := []float64{0.5, 200} // 200 should normalize to (200-100)/(200-100) = 1.0
	expectedEdge2 := []float64{0.5, 1.0}
	normalizedEdge2 := nn.applyNormalizations(edgeInputs2)
	assert.Equal(t, expectedEdge2, normalizedEdge2, "Normalization should handle max value correctly")
}

func TestShallowNeuralNetwork_ExportImport_WithNormalizations(t *testing.T) {
	// Create and configure original network
	nn := NewShallowNeuralNetwork(3, 4, nil)
	err := nn.SetFeatureNormalization(0, NormalizationMinMax, 100, 200, 0, 0)
	require.NoError(t, err)
	err = nn.SetFeatureNormalization(2, NormalizationZScore, 0, 0, 50, 10)
	require.NoError(t, err)

	// Export
	exported := nn.Export()
	require.NotNil(t, exported)

	// Import into new network
	nn2 := &ShallowNeuralNetwork{}
	err = nn2.Import(exported)
	require.NoError(t, err)

	// Verify normalizations were preserved
	assert.Len(t, nn2.featureNormalizations, 2)

	// Find each normalization (order may vary)
	var minMaxNorm, zScoreNorm *FeatureNormalization
	for _, fn := range nn2.featureNormalizations {
		switch fn.FeatureIndex {
		case 0:
			minMaxNorm = fn
		case 2:
			zScoreNorm = fn
		}
	}

	require.NotNil(t, minMaxNorm)
	assert.Equal(t, NormalizationMinMax, minMaxNorm.Type)
	assert.Equal(t, 100.0, minMaxNorm.Min)
	assert.Equal(t, 200.0, minMaxNorm.Max)

	require.NotNil(t, zScoreNorm)
	assert.Equal(t, NormalizationZScore, zScoreNorm.Type)
	assert.Equal(t, 50.0, zScoreNorm.Mean)
	assert.Equal(t, 10.0, zScoreNorm.StdDev)
}

func TestShallowNeuralNetwork_Normalization_EndToEnd(t *testing.T) {
	// Create network with queue length normalization (simulating real use case)
	nn := NewShallowNeuralNetwork(9, 6, nil)

	minQueue := 50000.0
	maxQueue := 200000.0

	// Train with pre-normalized data (matching production workflow)
	samples := []TrainingSample{
		// Low queue (pre-normalized): should scale
		{Inputs: []float64{0.15, 0.15, 0.0, 0.0, 0.5, (150000 - minQueue) / (maxQueue - minQueue), 0.3, 0.5, 0.5}, Expected: []float64{1.0}},
		// High queue (pre-normalized): should not scale
		{Inputs: []float64{0.9, 0.9, 0.0, 0.0, 0.9, (60000 - minQueue) / (maxQueue - minQueue), 0.0, 0.9, 0.9}, Expected: []float64{0.0}},
	}
	err := nn.Train(samples, 1000, ScaleUpThresholdDefault)
	require.NoError(t, err)

	// After training, configure normalization for production use
	err = nn.SetFeatureNormalization(5, NormalizationMinMax, minQueue, maxQueue, 0, 0)
	require.NoError(t, err)

	// Export and re-import to ensure normalization persists
	exported := nn.Export()
	nn2 := &ShallowNeuralNetwork{}
	err = nn2.Import(exported)
	require.NoError(t, err)

	// Predict with raw queue values (normalization applied automatically)
	lowQueueInputs := []float64{0.15, 0.15, 0.0, 0.0, 0.5, 150000, 0.3, 0.5, 0.5}
	highQueueInputs := []float64{0.9, 0.9, 0.0, 0.0, 0.9, 60000, 0.0, 0.9, 0.9}

	lowPrediction := nn2.PredictSample(lowQueueInputs)
	highPrediction := nn2.PredictSample(highQueueInputs)

	// Low queue should predict scale (closer to 1)
	// High queue should predict no scale (closer to 0)
	assert.True(t, lowPrediction[0] > highPrediction[0],
		"Low queue (%f) should have higher prediction than high queue (%f)", lowPrediction[0], highPrediction[0])
}

// TestShallowNeuralNetwork_PredictionVariability tests that the improved neural network
// produces varied predictions rather than saturated values close to 0 or 1
func TestShallowNeuralNetwork_PredictionVariability(t *testing.T) {
	// Create a network with sufficient capacity
	nn := NewShallowNeuralNetwork(3, 8, nil)

	// Create varied training data representing different resource scenarios
	samples := []TrainingSample{
		// Very low resources - should scale up
		{Inputs: []float64{0.1, 0.15, 0.05}, Expected: []float64{0.9}},
		{Inputs: []float64{0.05, 0.1, 0.08}, Expected: []float64{0.85}},

		// Medium-low resources - should scale up moderately
		{Inputs: []float64{0.3, 0.35, 0.25}, Expected: []float64{0.7}},
		{Inputs: []float64{0.4, 0.3, 0.35}, Expected: []float64{0.65}},

		// Medium resources - neutral scaling
		{Inputs: []float64{0.5, 0.55, 0.5}, Expected: []float64{0.5}},
		{Inputs: []float64{0.55, 0.5, 0.45}, Expected: []float64{0.45}},

		// Medium-high resources - should scale down moderately
		{Inputs: []float64{0.7, 0.65, 0.75}, Expected: []float64{0.35}},
		{Inputs: []float64{0.65, 0.7, 0.6}, Expected: []float64{0.3}},

		// High resources - should scale down
		{Inputs: []float64{0.85, 0.9, 0.88}, Expected: []float64{0.15}},
		{Inputs: []float64{0.9, 0.95, 0.85}, Expected: []float64{0.1}},
	}

	// Train with moderate learning rate and more epochs for better convergence
	err := nn.Train(samples, 2000, 0.3)
	require.NoError(t, err)

	// Test predictions across the spectrum
	testCases := []struct {
		name     string
		inputs   []float64
		expected string
	}{
		{"Low resources", []float64{0.1, 0.1, 0.1}, "high scaling"},
		{"Medium-low resources", []float64{0.3, 0.3, 0.3}, "moderate scaling"},
		{"Medium resources", []float64{0.5, 0.5, 0.5}, "neutral scaling"},
		{"Medium-high resources", []float64{0.7, 0.7, 0.7}, "moderate reduction"},
		{"High resources", []float64{0.9, 0.9, 0.9}, "low scaling"},
	}

	predictions := make([]float64, len(testCases))

	t.Logf("Prediction results with improved neural network:")
	for i, tc := range testCases {
		prediction := nn.PredictSample(tc.inputs)[0]
		predictions[i] = prediction
		t.Logf("  %s: %.4f (%s)", tc.name, prediction, tc.expected)
	}

	// Verify predictions are not saturated (not too close to 0 or 1)
	for i, pred := range predictions {
		assert.True(t, pred > 0.05 && pred < 0.95,
			"Prediction %d (%f) should be between 0.05 and 0.95 to avoid saturation", i, pred)
	}

	// Verify predictions show appropriate ordering (low resources -> higher scaling)
	assert.True(t, predictions[0] > predictions[1], "Low resources should predict higher scaling than medium-low")
	assert.True(t, predictions[1] > predictions[2], "Medium-low resources should predict higher scaling than medium")
	assert.True(t, predictions[2] > predictions[3], "Medium resources should predict higher scaling than medium-high")
	assert.True(t, predictions[3] > predictions[4], "Medium-high resources should predict higher scaling than high")

	// Verify there's sufficient variation (not all clustered near 0.5)
	var variance float64
	mean := 0.0
	for _, pred := range predictions {
		mean += pred
	}
	mean /= float64(len(predictions))

	for _, pred := range predictions {
		diff := pred - mean
		variance += diff * diff
	}
	variance /= float64(len(predictions))

	assert.True(t, variance > 0.01, "Predictions should show sufficient variance (got %f)", variance)
	t.Logf("Prediction variance: %.4f (good variance indicates non-saturated behavior)", variance)
}

func TestShallowNeuralNetwork_SetFeatureDescriptors(t *testing.T) {
	t.Run("nil descriptors returns error", func(t *testing.T) {
		nn := NewShallowNeuralNetwork(3, 2, nil)
		err := nn.SetFeatureDescriptors(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("valid descriptors are stored", func(t *testing.T) {
		nn := NewShallowNeuralNetwork(3, 2, nil)
		descriptors := []*FeatureDescriptor{
			{SourceMetricName: "cpu", UseTrend: false},
			{SourceMetricName: "mem", UseTrend: true},
		}
		err := nn.SetFeatureDescriptors(descriptors)
		require.NoError(t, err)
		assert.Equal(t, descriptors, nn.ListFeatureDescriptors())
	})

	t.Run("replaces existing descriptors", func(t *testing.T) {
		nn := NewShallowNeuralNetwork(3, 2, nil)
		first := []*FeatureDescriptor{{SourceMetricName: "cpu", UseTrend: false}}
		_ = nn.SetFeatureDescriptors(first)
		second := []*FeatureDescriptor{{SourceMetricName: "mem", UseTrend: true}}
		_ = nn.SetFeatureDescriptors(second)
		assert.Equal(t, second, nn.ListFeatureDescriptors())
	})
}

func TestShallowNeuralNetwork_ListFeatureDescriptors(t *testing.T) {
	t.Run("returns nil when none set", func(t *testing.T) {
		nn := NewShallowNeuralNetwork(3, 2, nil)
		assert.Nil(t, nn.ListFeatureDescriptors())
	})

	t.Run("returns descriptors after set", func(t *testing.T) {
		nn := NewShallowNeuralNetwork(3, 2, nil)
		descriptors := []*FeatureDescriptor{
			{SourceMetricName: "cpu", UseTrend: false},
		}
		_ = nn.SetFeatureDescriptors(descriptors)
		assert.Equal(t, descriptors, nn.ListFeatureDescriptors())
	})
}

func TestShallowNeuralNetwork_SetFeatureNormalizations(t *testing.T) {
	t.Run("applies all normalizations", func(t *testing.T) {
		nn := NewShallowNeuralNetwork(3, 2, nil)
		norms := []*FeatureNormalization{
			{FeatureIndex: 0, Type: NormalizationMinMax, Min: 0, Max: 100},
			{FeatureIndex: 1, Type: NormalizationMinMaxSymmetric, Min: -50, Max: 50},
		}
		err := nn.SetFeatureNormalizations(norms)
		require.NoError(t, err)
		assert.Len(t, nn.featureNormalizations, 2)
	})

	t.Run("returns error on invalid normalization entry", func(t *testing.T) {
		nn := NewShallowNeuralNetwork(3, 2, nil)
		norms := []*FeatureNormalization{
			{FeatureIndex: 0, Type: NormalizationMinMax, Min: 100, Max: 0}, // min > max: invalid
		}
		err := nn.SetFeatureNormalizations(norms)
		require.Error(t, err)
	})

	t.Run("empty slice is a no-op", func(t *testing.T) {
		nn := NewShallowNeuralNetwork(3, 2, nil)
		err := nn.SetFeatureNormalizations([]*FeatureNormalization{})
		require.NoError(t, err)
		assert.Nil(t, nn.featureNormalizations)
	})
}
