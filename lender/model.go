package lender

import (
	"github.com/sassoftware/gopher-hole/metrics"
)

// NormalizationType specifies the type of normalization to apply to a feature.
type NormalizationType string

const (
	// NormalizationMinMax scales features to a [0, 1] range using min-max normalization.
	NormalizationMinMax NormalizationType = "minmax"
	// NormalizationMinMaxSymmetric scales features to a [-1, 1] range using symmetric min-max normalization.
	NormalizationMinMaxSymmetric NormalizationType = "minmax_symmetric"
	// NormalizationZScore standardizes features using z-score normalization (mean=0, stddev=1).
	NormalizationZScore NormalizationType = "zscore"
)

// FeatureNormalization stores normalization parameters for a specific feature.
// The model uses these parameters to automatically normalize inputs during prediction.
type FeatureNormalization struct {
	FeatureIndex int               `json:"featureIndex"`     // Zero-based index of the feature
	Type         NormalizationType `json:"type"`             // Type of normalization to apply
	Min          float64           `json:"min,omitempty"`    // Minimum value for min-max normalization
	Max          float64           `json:"max,omitempty"`    // Maximum value for min-max normalization
	Mean         float64           `json:"mean,omitempty"`   // Mean value for z-score normalization
	StdDev       float64           `json:"stdDev,omitempty"` // Standard deviation for z-score normalization
}

// ApplyNormalization applies the normalization described by fn to val and returns
// the scaled result. It is the single canonical implementation of the normalization
// math shared by training (LoadTrainingData) and inference (ShallowNeuralNetwork).
func ApplyNormalization(val float64, fn *FeatureNormalization) float64 {
	switch fn.Type {
	case NormalizationMinMax:
		if fn.Max == fn.Min {
			return 0.0
		}
		return (val - fn.Min) / (fn.Max - fn.Min)
	case NormalizationMinMaxSymmetric:
		if fn.Max == fn.Min {
			return 0.0
		}
		return 2*(val-fn.Min)/(fn.Max-fn.Min) - 1
	case NormalizationZScore:
		if fn.StdDev == 0 {
			return 0.0
		}
		return (val - fn.Mean) / fn.StdDev
	default:
		return val
	}
}

// FeatureDescriptor describes how a single neural network input is derived
// from a base metric. It is set imperatively at model load time and is NOT
// serialized to the model JSON file.
//
// FeatureDescriptor is the single source of truth for a feature's slot in the
// input vector (InputIndex) and the normalization applied to it (NormType).
// Training and inference both derive this information from the descriptor list.
type FeatureDescriptor struct {
	// SourceMetricName is the name of the base *metrics.Metric to read.
	SourceMetricName string
	// UseTrend, when true, calls Trend() on the metric instead of GetMostRecent().
	UseTrend bool
	// InputIndex is the zero-based position in the neural-network input vector that
	// this descriptor feeds. Explicit indexing means descriptor order is irrelevant.
	InputIndex int
	// NormType is the normalization applied to this feature before it reaches the
	// network. An empty value means the raw value is passed through unchanged.
	NormType NormalizationType
}

// An AIModel is responsible for accepting a list of metrics and making a
// prediction. Returning the prediction so the lender can make a decision.
// It is assumed that the model is already trained prior to registering with
// the lender.
type AIModel interface {
	ListMetrics() []*metrics.Metric
	Predict([]*metrics.Metric) (float64, error)
}

// TrainingSample holds training data for a shallow neural network.
// It contains input features and the expected output for supervised learning.
type TrainingSample struct {
	Inputs   []float64
	Expected []float64
}

// ModelEval contains evaluation metrics for a trained model.
type ModelEval struct {
	Accuracy  float64
	Loss      float64
	Precision float64
	Recall    float64
}

// Trainable defines the interface for models that support training,
// import/export of weights, and evaluation capabilities.
type Trainable interface {
	// Import loads model weights and configuration from serialized bytes.
	Import(data []byte) error
	// Export serializes the model weights and configuration to bytes.
	Export() []byte
	// Evaluate assesses the model performance against test samples.
	// Returns evaluation metrics if the model meets the threshold criteria.
	Evaluate(samples []TrainingSample, threshold float64) ModelEval
	// Train updates the model weights using the provided training samples.
	Train(samples []TrainingSample, epochs int, learningRate float64) error
}

// Predictor defines the prediction capability that BaseModel requires
// for evaluation. Implementations must provide this method.
type Predictor interface {
	// PredictSample returns the model's prediction for a single input sample.
	PredictSample(inputs []float64) []float64
}

// BaseModel provides a generic implementation of ListMetrics and Evaluate
// that can be embedded in concrete model implementations.
type BaseModel struct {
	Metrics   []*metrics.Metric
	Predictor Predictor
}

// ListMetrics returns the list of metrics associated with this model.
func (b *BaseModel) ListMetrics() []*metrics.Metric {
	return b.Metrics
}

// Evaluate assesses the model performance against test samples.
// It calculates accuracy, loss, precision, and recall based on the threshold.
func (b *BaseModel) Evaluate(samples []TrainingSample, threshold float64) ModelEval {
	if len(samples) == 0 || b.Predictor == nil {
		return ModelEval{}
	}

	var totalLoss float64
	var truePositives, falsePositives, trueNegatives, falseNegatives int

	for _, sample := range samples {
		predicted := b.Predictor.PredictSample(sample.Inputs)

		// Calculate mean squared error loss for this sample
		sampleLoss := 0.0
		for i := range predicted {
			if i < len(sample.Expected) {
				diff := predicted[i] - sample.Expected[i]
				sampleLoss += diff * diff
			}
		}
		if len(predicted) > 0 {
			sampleLoss /= float64(len(predicted))
		}
		totalLoss += sampleLoss

		// For classification metrics, use threshold on first output
		if len(predicted) > 0 && len(sample.Expected) > 0 {
			predictedClass := predicted[0] >= threshold
			expectedClass := sample.Expected[0] >= threshold

			switch {
			case predictedClass && expectedClass:
				truePositives++
			case predictedClass && !expectedClass:
				falsePositives++
			case !predictedClass && !expectedClass:
				trueNegatives++
			case !predictedClass && expectedClass:
				falseNegatives++
			}
		}
	}

	total := truePositives + trueNegatives + falsePositives + falseNegatives
	accuracy := 0.0
	if total > 0 {
		accuracy = float64(truePositives+trueNegatives) / float64(total)
	}

	precision := 0.0
	if truePositives+falsePositives > 0 {
		precision = float64(truePositives) / float64(truePositives+falsePositives)
	}

	recall := 0.0
	if truePositives+falseNegatives > 0 {
		recall = float64(truePositives) / float64(truePositives+falseNegatives)
	}

	avgLoss := totalLoss / float64(len(samples))

	return ModelEval{
		Accuracy:  accuracy,
		Loss:      avgLoss,
		Precision: precision,
		Recall:    recall,
	}
}
