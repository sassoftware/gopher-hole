package lender

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/sassoftware/gopher-hole/metrics"
)

const (
	// Training early stopping parameters
	trainingPatienceEpochs         = 500    // Number of epochs to wait for improvement
	trainingMinDelta               = 0.0001 // Minimum change in loss to qualify as improvement
	trainingEvaluationPeriodEpochs = 100    // Evaluate every N epochs

	// Sigmoid saturation constants - prevent extreme values that cause saturation
	sigmoidSaturationHigh = 500.0  // Upper bound for sigmoid input to prevent saturation at 1
	sigmoidSaturationLow  = -500.0 // Lower bound for sigmoid input to prevent saturation at 0

	// Training regularization parameters
	defaultLearningRateDecay = 0.999  // Learning rate decay per epoch
	defaultWeightDecay       = 0.0001 // L2 regularization strength
	defaultGradientClipValue = 1.0    // Gradient clipping threshold
)

// ShallowNeuralNetwork implements a single hidden layer neural network
// that satisfies both AIModel and Trainable interfaces.
type ShallowNeuralNetwork struct {
	BaseModel

	// Network architecture
	numInputs  int
	numHidden  int
	numOutputs int

	// Weights and biases
	weightsInputHidden  [][]float64 // [numInputs][numHidden]
	biasesHidden        []float64   // [numHidden]
	weightsHiddenOutput [][]float64 // [numHidden][numOutputs]
	biasesOutput        []float64   // [numOutputs]

	// Feature normalization
	featureNormalizations []*FeatureNormalization // Normalization parameters for input features

	// Feature descriptors describe how each input is derived from a base metric.
	// Set imperatively at load time; not serialized to the model JSON.
	featureDescriptors []*FeatureDescriptor
}

// shallowNeuralNetworkData is used for JSON serialization of the network.
// This format is designed to be compatible with Java deserialization.
type shallowNeuralNetworkData struct {
	// Model metadata
	ModelType   string `json:"modelType"`
	Version     string `json:"version"`
	CreatedAt   string `json:"createdAt,omitempty"`
	Description string `json:"description,omitempty"`

	// Architecture
	NumInputs  int `json:"numInputs"`
	NumHidden  int `json:"numHidden"`
	NumOutputs int `json:"numOutputs"`

	// Weights and biases
	WeightsInputHidden  [][]float64 `json:"weightsInputHidden"`
	BiasesHidden        []float64   `json:"biasesHidden"`
	WeightsHiddenOutput [][]float64 `json:"weightsHiddenOutput"`
	BiasesOutput        []float64   `json:"biasesOutput"`

	// Activation functions (for compatibility)
	HiddenActivation string `json:"hiddenActivation"`
	OutputActivation string `json:"outputActivation"`

	// Feature normalization
	FeatureNormalizations []*FeatureNormalization `json:"featureNormalizations,omitempty"`
}

// NewShallowNeuralNetwork creates a new shallow neural network with the specified
// number of inputs and hidden neurons. The network has a single output neuron
// suitable for binary classification or regression.
func NewShallowNeuralNetwork(numInputs, numHidden int, metricsList []*metrics.Metric) *ShallowNeuralNetwork {
	return NewShallowNeuralNetworkWithOutputs(numInputs, numHidden, 1, metricsList)
}

// NewShallowNeuralNetworkWithOutputs creates a new shallow neural network with
// configurable number of inputs, hidden neurons, and outputs.
func NewShallowNeuralNetworkWithOutputs(numInputs, numHidden, numOutputs int, metricsList []*metrics.Metric) *ShallowNeuralNetwork {
	nn := &ShallowNeuralNetwork{
		numInputs:  numInputs,
		numHidden:  numHidden,
		numOutputs: numOutputs,
	}

	// Initialize weights with Xavier/Glorot initialization
	nn.initializeWeights()

	// Set up BaseModel
	nn.Metrics = metricsList
	nn.Predictor = nn

	return nn
}

// initializeWeights initializes the network weights using Xavier/Glorot initialization
// which helps with gradient flow during training.
func (nn *ShallowNeuralNetwork) initializeWeights() {
	// Initialize input -> hidden weights
	nn.weightsInputHidden = make([][]float64, nn.numInputs)
	scaleInputHidden := math.Sqrt(2.0 / float64(nn.numInputs+nn.numHidden))
	for i := 0; i < nn.numInputs; i++ {
		nn.weightsInputHidden[i] = make([]float64, nn.numHidden)
		for j := 0; j < nn.numHidden; j++ {
			nn.weightsInputHidden[i][j] = rand.NormFloat64() * scaleInputHidden //nolint:gosec // disable G404
		}
	}

	// Initialize hidden biases
	nn.biasesHidden = make([]float64, nn.numHidden)

	// Initialize hidden -> output weights
	nn.weightsHiddenOutput = make([][]float64, nn.numHidden)
	scaleHiddenOutput := math.Sqrt(2.0 / float64(nn.numHidden+nn.numOutputs))
	for i := 0; i < nn.numHidden; i++ {
		nn.weightsHiddenOutput[i] = make([]float64, nn.numOutputs)
		for j := 0; j < nn.numOutputs; j++ {
			nn.weightsHiddenOutput[i][j] = rand.NormFloat64() * scaleHiddenOutput //nolint:gosec // disable G404
		}
	}

	// Initialize output biases
	nn.biasesOutput = make([]float64, nn.numOutputs)
}

// sigmoid activation function
func sigmoid(x float64) float64 {
	// Clamp extreme values to prevent overflow and saturation
	if x > sigmoidSaturationHigh {
		return 1.0
	}
	if x < sigmoidSaturationLow {
		return 0.0
	}
	return 1.0 / (1.0 + math.Exp(-x))
}

// sigmoidDerivative returns the derivative of sigmoid given the sigmoid output
func sigmoidDerivative(sigmoidOutput float64) float64 {
	return sigmoidOutput * (1.0 - sigmoidOutput)
}

// forward performs forward propagation and returns hidden activations and outputs
func (nn *ShallowNeuralNetwork) forward(inputs []float64) (hiddenActivations, outputs []float64) { //nolint:gocognit
	// Calculate hidden layer activations
	hiddenActivations = make([]float64, nn.numHidden)
	for j := 0; j < nn.numHidden; j++ {
		sum := nn.biasesHidden[j]
		for i := 0; i < nn.numInputs && i < len(inputs); i++ {
			product := inputs[i] * nn.weightsInputHidden[i][j]
			// Check for overflow/underflow before adding to sum
			if math.IsInf(product, 0) || math.IsNaN(product) {
				// If product is infinite/NaN, clamp the sum to prevent propagation
				if product > 0 {
					sum = sigmoidSaturationHigh // Will saturate sigmoid to 1.0
				} else {
					sum = sigmoidSaturationLow // Will saturate sigmoid to 0.0
				}
				break // No need to continue with this neuron
			}
			sum += product
			// Also check if sum itself becomes infinite
			if math.IsInf(sum, 0) {
				if sum > 0 {
					sum = sigmoidSaturationHigh
				} else {
					sum = sigmoidSaturationLow
				}
				break
			}
		}
		hiddenActivations[j] = sigmoid(sum)
	}

	// Calculate output layer activations
	outputs = make([]float64, nn.numOutputs)
	for k := 0; k < nn.numOutputs; k++ {
		sum := nn.biasesOutput[k]
		for j := 0; j < nn.numHidden; j++ {
			product := hiddenActivations[j] * nn.weightsHiddenOutput[j][k]
			// Check for overflow/underflow before adding to sum
			if math.IsInf(product, 0) || math.IsNaN(product) {
				// If product is infinite/NaN, clamp the sum to prevent propagation
				if product > 0 {
					sum = sigmoidSaturationHigh // Will saturate sigmoid to 1.0
				} else {
					sum = sigmoidSaturationLow // Will saturate sigmoid to 0.0
				}
				break // No need to continue with this neuron
			}
			sum += product
			// Also check if sum itself becomes infinite
			if math.IsInf(sum, 0) {
				if sum > 0 {
					sum = sigmoidSaturationHigh
				} else {
					sum = sigmoidSaturationLow
				}
				break
			}
		}
		outputs[k] = sigmoid(sum)
	}

	return hiddenActivations, outputs
}

// PredictSample implements the Predictor interface for BaseModel.Evaluate.
// It applies any configured normalizations before forwarding the inputs.
func (nn *ShallowNeuralNetwork) PredictSample(inputs []float64) []float64 {
	// Apply normalizations if configured
	normalizedInputs := nn.applyNormalizations(inputs)
	_, outputs := nn.forward(normalizedInputs)
	return outputs
}

// Predict implements AIModel interface.
// It extracts values from the metrics and returns a single prediction.
func (nn *ShallowNeuralNetwork) Predict(metricsList []*metrics.Metric) (float64, error) {
	if metricsList == nil {
		return 0.0, fmt.Errorf("metricsList cannot be nil")
	}

	// Extract metric values as inputs
	inputs := make([]float64, 0, len(metricsList))
	for _, m := range metricsList {
		if m != nil {
			// Use the metric's most recent value
			if value, err := m.GetMostRecent(5 * time.Minute); err == nil {
				inputs = append(inputs, value)
			} else {
				// If we can't get the value, return an error
				return 0.0, fmt.Errorf("failed to get most recent metric value: %w", err)
			}
		}
	}

	// Pad or truncate to match expected inputs
	for len(inputs) < nn.numInputs {
		inputs = append(inputs, 0.0)
	}
	if len(inputs) > nn.numInputs {
		inputs = inputs[:nn.numInputs]
	}

	outputs := nn.PredictSample(inputs)
	if len(outputs) > 0 {
		return outputs[0], nil
	}
	return 0.0, nil
}

// Train implements Trainable interface.
// It uses stochastic gradient descent with backpropagation.
func (nn *ShallowNeuralNetwork) Train(samples []TrainingSample, epochs int, learningRate float64) error { //nolint:gocognit
	if len(samples) == 0 {
		return fmt.Errorf("training samples cannot be empty")
	}
	if epochs <= 0 {
		return fmt.Errorf("epochs must be positive, got %d", epochs)
	}
	if learningRate <= 0 {
		return fmt.Errorf("learning rate must be positive, got %f", learningRate)
	}

	// Validate training samples
	for i, sample := range samples {
		if len(sample.Inputs) != nn.numInputs {
			return fmt.Errorf("training sample %d has %d inputs, expected %d", i, len(sample.Inputs), nn.numInputs)
		}
		if len(sample.Expected) != nn.numOutputs {
			return fmt.Errorf("training sample %d has %d outputs, expected %d", i, len(sample.Expected), nn.numOutputs)
		}
	}

	bestLoss := math.Inf(1)
	epochsWithoutImprovement := 0
	currentLearningRate := learningRate

	for epoch := 0; epoch < epochs; epoch++ {
		// Apply learning rate decay to prevent overshooting
		if epoch > 0 {
			currentLearningRate = learningRate * math.Pow(defaultLearningRateDecay, float64(epoch))
		}

		// Shuffle samples each epoch for better convergence
		shuffled := make([]TrainingSample, len(samples))
		copy(shuffled, samples)
		rand.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		for _, sample := range shuffled {
			nn.trainSample(sample, currentLearningRate)
		}

		// Apply L2 regularization (weight decay) to prevent overfitting
		nn.applyWeightDecay(defaultWeightDecay)

		// Early stopping evaluation
		if epoch%trainingEvaluationPeriodEpochs == 0 && epoch > 0 {
			// Calculate current loss on all samples
			currentLoss := 0.0
			for _, sample := range samples {
				predicted := nn.PredictSample(sample.Inputs)
				for i := range predicted {
					if i < len(sample.Expected) {
						diff := predicted[i] - sample.Expected[i]
						currentLoss += diff * diff
					}
				}
			}
			currentLoss /= float64(len(samples))

			// Check if we've improved
			if bestLoss-currentLoss > trainingMinDelta {
				bestLoss = currentLoss
				epochsWithoutImprovement = 0
			} else {
				epochsWithoutImprovement++
			}

			// Exit early if no improvement for patience epochs
			if epochsWithoutImprovement >= trainingPatienceEpochs/trainingEvaluationPeriodEpochs {
				break
			}
		}
	}

	return nil
}

// trainSample performs one step of backpropagation for a single sample
func (nn *ShallowNeuralNetwork) trainSample(sample TrainingSample, learningRate float64) {
	// Forward pass
	hiddenActivations, outputs := nn.forward(sample.Inputs)

	// Calculate output layer errors (delta) with gradient clipping
	outputDeltas := make([]float64, nn.numOutputs)
	for k := 0; k < nn.numOutputs; k++ {
		expected := 0.0
		if k < len(sample.Expected) {
			expected = sample.Expected[k]
		}
		res := expected - outputs[k]
		delta := res * sigmoidDerivative(outputs[k])
		// Clip gradients to prevent exploding gradients
		if delta > defaultGradientClipValue {
			delta = defaultGradientClipValue
		} else if delta < -defaultGradientClipValue {
			delta = -defaultGradientClipValue
		}
		outputDeltas[k] = delta
	}

	// Calculate hidden layer errors (delta) with gradient clipping
	hiddenDeltas := make([]float64, nn.numHidden)
	for j := 0; j < nn.numHidden; j++ {
		res := 0.0
		for k := 0; k < nn.numOutputs; k++ {
			res += outputDeltas[k] * nn.weightsHiddenOutput[j][k]
		}
		delta := res * sigmoidDerivative(hiddenActivations[j])
		// Clip gradients to prevent exploding gradients
		if delta > defaultGradientClipValue {
			delta = defaultGradientClipValue
		} else if delta < -defaultGradientClipValue {
			delta = -defaultGradientClipValue
		}
		hiddenDeltas[j] = delta
	}

	// Update hidden -> output weights and biases
	for j := 0; j < nn.numHidden; j++ {
		for k := 0; k < nn.numOutputs; k++ {
			nn.weightsHiddenOutput[j][k] += learningRate * outputDeltas[k] * hiddenActivations[j]
		}
	}
	for k := 0; k < nn.numOutputs; k++ {
		nn.biasesOutput[k] += learningRate * outputDeltas[k]
	}

	// Update input -> hidden weights and biases
	for i := 0; i < nn.numInputs && i < len(sample.Inputs); i++ {
		for j := 0; j < nn.numHidden; j++ {
			nn.weightsInputHidden[i][j] += learningRate * hiddenDeltas[j] * sample.Inputs[i]
		}
	}
	for j := 0; j < nn.numHidden; j++ {
		nn.biasesHidden[j] += learningRate * hiddenDeltas[j]
	}
}

// applyWeightDecay applies L2 regularization to all weights to prevent overfitting
func (nn *ShallowNeuralNetwork) applyWeightDecay(weightDecay float64) {
	// Apply weight decay to input -> hidden weights
	for i := 0; i < nn.numInputs; i++ {
		for j := 0; j < nn.numHidden; j++ {
			nn.weightsInputHidden[i][j] *= (1.0 - weightDecay)
		}
	}

	// Apply weight decay to hidden -> output weights
	for j := 0; j < nn.numHidden; j++ {
		for k := 0; k < nn.numOutputs; k++ {
			nn.weightsHiddenOutput[j][k] *= (1.0 - weightDecay)
		}
	}

	// Note: Biases typically don't get regularized as they don't contribute to overfitting
}

// Import implements Trainable interface.
// It loads model weights and configuration from JSON bytes.
// Compatible with Java-serialized models that follow the same schema.
func (nn *ShallowNeuralNetwork) Import(data []byte) error {
	var imported shallowNeuralNetworkData
	if err := json.Unmarshal(data, &imported); err != nil {
		return err
	}

	// Validate model type if present
	if imported.ModelType != "" && imported.ModelType != "ShallowNeuralNetwork" {
		return fmt.Errorf("incompatible model type: expected ShallowNeuralNetwork, got %s", imported.ModelType)
	}

	// Validate architecture dimensions
	if imported.NumInputs <= 0 || imported.NumHidden <= 0 || imported.NumOutputs <= 0 {
		return fmt.Errorf("invalid architecture dimensions: inputs=%d, hidden=%d, outputs=%d",
			imported.NumInputs, imported.NumHidden, imported.NumOutputs)
	}

	// Validate weights dimensions
	if len(imported.WeightsInputHidden) != imported.NumInputs {
		return fmt.Errorf("weightsInputHidden dimension mismatch: expected %d, got %d",
			imported.NumInputs, len(imported.WeightsInputHidden))
	}
	if len(imported.BiasesHidden) != imported.NumHidden {
		return fmt.Errorf("biasesHidden dimension mismatch: expected %d, got %d",
			imported.NumHidden, len(imported.BiasesHidden))
	}
	if len(imported.WeightsHiddenOutput) != imported.NumHidden {
		return fmt.Errorf("weightsHiddenOutput dimension mismatch: expected %d, got %d",
			imported.NumHidden, len(imported.WeightsHiddenOutput))
	}
	if len(imported.BiasesOutput) != imported.NumOutputs {
		return fmt.Errorf("biasesOutput dimension mismatch: expected %d, got %d",
			imported.NumOutputs, len(imported.BiasesOutput))
	}

	// Validate activation functions if specified (Java models may include these)
	if imported.HiddenActivation != "" && imported.HiddenActivation != "sigmoid" {
		return fmt.Errorf("unsupported hidden activation: %s", imported.HiddenActivation)
	}
	if imported.OutputActivation != "" && imported.OutputActivation != "sigmoid" {
		return fmt.Errorf("unsupported output activation: %s", imported.OutputActivation)
	}

	nn.numInputs = imported.NumInputs
	nn.numHidden = imported.NumHidden
	nn.numOutputs = imported.NumOutputs
	nn.weightsInputHidden = imported.WeightsInputHidden
	nn.biasesHidden = imported.BiasesHidden
	nn.weightsHiddenOutput = imported.WeightsHiddenOutput
	nn.biasesOutput = imported.BiasesOutput
	nn.featureNormalizations = imported.FeatureNormalizations

	return nil
}

// Export implements Trainable interface.
// It serializes the model weights and configuration to JSON bytes in a format
// compatible with Java deserialization (follows JavaBean conventions).
func (nn *ShallowNeuralNetwork) Export() []byte {
	data := shallowNeuralNetworkData{
		ModelType:             "ShallowNeuralNetwork",
		Version:               "1.0",
		CreatedAt:             time.Now().UTC().Format(time.RFC3339),
		Description:           fmt.Sprintf("%d-input, %d-hidden, %d-output neural network", nn.numInputs, nn.numHidden, nn.numOutputs),
		NumInputs:             nn.numInputs,
		NumHidden:             nn.numHidden,
		NumOutputs:            nn.numOutputs,
		WeightsInputHidden:    nn.weightsInputHidden,
		BiasesHidden:          nn.biasesHidden,
		WeightsHiddenOutput:   nn.weightsHiddenOutput,
		BiasesOutput:          nn.biasesOutput,
		HiddenActivation:      "sigmoid",
		OutputActivation:      "sigmoid",
		FeatureNormalizations: nn.featureNormalizations,
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil
	}
	return bytes
}

// GetArchitecture returns the network architecture details.
func (nn *ShallowNeuralNetwork) GetArchitecture() (inputs, hidden, outputs int) {
	return nn.numInputs, nn.numHidden, nn.numOutputs
}

// SetFeatureNormalization configures normalization for a specific feature.
// The normalization will be automatically applied during prediction.
// For min-max normalization, provide min and max values.
// For z-score normalization, provide mean and stdDev.
func (nn *ShallowNeuralNetwork) SetFeatureNormalization(featureIndex int, normType NormalizationType, lower, upper, mean, stdDev float64) error {
	if featureIndex < 0 || featureIndex >= nn.numInputs {
		return fmt.Errorf("feature index %d out of range [0, %d)", featureIndex, nn.numInputs)
	}

	// Validate normalization parameters
	if (normType == NormalizationMinMax || normType == NormalizationMinMaxSymmetric) && lower >= upper {
		return fmt.Errorf("min-max normalization requires min < max, got min=%f, max=%f", lower, upper)
	}
	if normType == NormalizationZScore && stdDev <= 0 {
		return fmt.Errorf("z-score normalization requires stdDev > 0, got stdDev=%f", stdDev)
	}

	// Initialize the slice if needed
	if nn.featureNormalizations == nil {
		nn.featureNormalizations = make([]*FeatureNormalization, 0)
	}

	// Remove existing normalization for this feature if present
	for i, fn := range nn.featureNormalizations {
		if fn.FeatureIndex == featureIndex {
			nn.featureNormalizations = append(nn.featureNormalizations[:i], nn.featureNormalizations[i+1:]...)
			break
		}
	}

	// Add new normalization
	nn.featureNormalizations = append(nn.featureNormalizations, &FeatureNormalization{
		FeatureIndex: featureIndex,
		Type:         normType,
		Min:          lower,
		Max:          upper,
		Mean:         mean,
		StdDev:       stdDev,
	})

	return nil
}

// SetFeatureDescriptors replaces the feature descriptor list in one call.
// Returns the first error encountered, if any.
func (nn *ShallowNeuralNetwork) SetFeatureDescriptors(descriptors []*FeatureDescriptor) error {
	if descriptors == nil {
		return fmt.Errorf("descriptors cannot be nil")
	}
	nn.featureDescriptors = descriptors
	return nil
}

// ListFeatureDescriptors returns the feature descriptor list.
func (nn *ShallowNeuralNetwork) ListFeatureDescriptors() []*FeatureDescriptor {
	return nn.featureDescriptors
}

// SetFeatureNormalizations replaces all feature normalizations in one call,
// delegating to SetFeatureNormalization for each entry.
// Returns the first error encountered, if any.
func (nn *ShallowNeuralNetwork) SetFeatureNormalizations(norms []*FeatureNormalization) error {
	for _, fn := range norms {
		if err := nn.SetFeatureNormalization(fn.FeatureIndex, fn.Type, fn.Min, fn.Max, fn.Mean, fn.StdDev); err != nil {
			return err
		}
	}
	return nil
}

// applyNormalizations applies configured normalizations to the input features.
// Returns a copy of the inputs with normalizations applied.
func (nn *ShallowNeuralNetwork) applyNormalizations(inputs []float64) []float64 {
	if len(nn.featureNormalizations) == 0 {
		return inputs
	}

	// Create a copy to avoid modifying the original
	normalized := make([]float64, len(inputs))
	copy(normalized, inputs)

	// Apply each normalization
	for _, fn := range nn.featureNormalizations {
		if fn.FeatureIndex >= 0 && fn.FeatureIndex < len(normalized) {
			normalized[fn.FeatureIndex] = ApplyNormalization(normalized[fn.FeatureIndex], fn)
		}
	}

	return normalized
}
