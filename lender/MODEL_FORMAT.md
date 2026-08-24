# Neural Network Model Export/Import Format

## Overview

The `ShallowNeuralNetwork` model uses a JSON format for serialization that is
compatible with Java deserialization. This allows models trained in Go to be
imported and used in Java applications, and vice versa.

## Export Format

When a model is exported using `Export()`, it produces JSON with the following structure:

```json
{
  "modelType": "ShallowNeuralNetwork",
  "version": "1.0",
  "createdAt": "2026-01-19T00:00:00Z",
  "description": "2-input, 4-hidden, 1-output neural network",
  "numInputs": 2,
  "numHidden": 4,
  "numOutputs": 1,
  "weightsInputHidden": [
    [0.123, 0.456, 0.789, 0.012],
    [0.345, 0.678, 0.901, 0.234]
  ],
  "biasesHidden": [0.1, 0.2, 0.3, 0.4],
  "weightsHiddenOutput": [
    [0.5],
    [0.6],
    [0.7],
    [0.8]
  ],
  "biasesOutput": [0.9],
  "hiddenActivation": "sigmoid",
  "outputActivation": "sigmoid"
}
```

## Field Descriptions

- **modelType**: Identifies the model type (always "ShallowNeuralNetwork")
- **version**: Format version for compatibility tracking
- **createdAt**: ISO 8601 timestamp of model export
- **description**: Human-readable description of architecture
- **numInputs**: Number of input neurons
- **numHidden**: Number of hidden layer neurons
- **numOutputs**: Number of output neurons
- **weightsInputHidden**: 2D array `[numInputs][numHidden]` of weights
- **biasesHidden**: 1D array [numHidden] of bias values
- **weightsHiddenOutput**: 2D array `[numHidden][numOutputs]` of weights
- **biasesOutput**: 1D array [numOutputs] of bias values
- **hiddenActivation**: Activation function for hidden layer (currently "sigmoid")
- **outputActivation**: Activation function for output layer (currently "sigmoid")

## Java Compatibility

### Import Validation

When importing a model, the following validations are performed:

1. **Model Type**: Ensures the model type matches "ShallowNeuralNetwork"
2. **Architecture Dimensions**: Validates positive values for inputs, hidden,
and outputs
3. **Weight Dimensions**: Verifies weight matrices match declared architecture
4. **Activation Functions**: Ensures only supported activations ("sigmoid") are
used

### Java Integration Example

A Java class for deserializing this format would look like:

```java
public class ShallowNeuralNetworkData {
    private String modelType;
    private String version;
    private String createdAt;
    private String description;
    private int numInputs;
    private int numHidden;
    private int numOutputs;
    private double[][] weightsInputHidden;
    private double[] biasesHidden;
    private double[][] weightsHiddenOutput;
    private double[] biasesOutput;
    private String hiddenActivation;
    private String outputActivation;

    // Getters and setters following JavaBean conventions
}
```

### Usage in Go

```go
// Create and train a model
nn := NewShallowNeuralNetwork(2, 4, metrics)
samples := []TrainingSample{
    {Inputs: []float64{0, 0}, Expected: []float64{0}},
    {Inputs: []float64{1, 1}, Expected: []float64{1}},
}
if err := nn.Train(samples, 1000, 1.0); err != nil {
    log.Fatalf("Failed to train model: %v", err)
}

// Export to JSON
exportedBytes := nn.Export()

// Save to file or send to Java application
err := os.WriteFile("model.json", exportedBytes, 0644)

// Later, import the model
nn2 := NewShallowNeuralNetwork(2, 4, metrics)
data, _ := os.ReadFile("model.json")
err = nn2.Import(data)
```

## Error Handling

The `Import()` method returns descriptive errors for common issues:

- Invalid JSON syntax
- Incompatible model type
- Invalid architecture dimensions
- Weight dimension mismatches
- Unsupported activation functions

## Version Compatibility

The format includes a version field to enable future enhancements while
maintaining backward compatibility. Current version is "1.0".

## Notes

- The export uses `json.MarshalIndent` for human-readable formatting
- All floating-point values preserve full precision
- Field names follow camelCase convention for Java compatibility
- Optional fields (createdAt, description) can be omitted during import
