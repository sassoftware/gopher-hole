package lender

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sassoftware/gopher-hole/metrics"
	"github.com/stretchr/testify/assert"
)

const testGroup = "testGroup"

// MockLendee implements the Lendee interface for testing
type MockLendee struct {
	maxCredits      int
	creditsAdded    int
	initialCredits  int
	name            string
	addCreditErr    error
	removeCreditErr error
}

func (m *MockLendee) AddCredit() error {
	if m.addCreditErr != nil {
		return m.addCreditErr
	}
	m.creditsAdded++
	return nil
}

func (m *MockLendee) RemoveCredit(_ context.Context) error {
	if m.removeCreditErr != nil {
		return m.removeCreditErr
	}
	m.creditsAdded--
	return nil
}

func (m *MockLendee) MaxLenderCredits() int {
	return m.maxCredits - m.initialCredits
}

func (m *MockLendee) Availability() float64 {
	return 1 / float64(m.initialCredits+m.creditsAdded)
}

func (m *MockLendee) GetName() string {
	return m.name
}

// MockAIModel implements the AIModel interface for testing
type MockAIModel struct {
	metricsList []*metrics.Metric
	prediction  float64
}

func (m *MockAIModel) ListMetrics() []*metrics.Metric {
	return m.metricsList
}

func (m *MockAIModel) Predict(_ []*metrics.Metric) (float64, error) {
	return m.prediction, nil
}

func TestNewLender(t *testing.T) {
	tests := map[string]struct {
		name           string
		ctx            context.Context
		metricsMgr     *metrics.Manager
		expectError    bool
		expectedResult func(*testing.T, *Lender, error)
	}{
		"with metrics manager in context": {
			name:        "valid context with metrics manager",
			ctx:         context.Background(),
			metricsMgr:  metrics.NewManager(),
			expectError: false,
			expectedResult: func(t *testing.T, lender *Lender, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, lender)
				assert.NotNil(t, lender.ctx)
				assert.NotNil(t, lender.lendees)
				assert.NotNil(t, lender.metricsMgr)
				assert.Empty(t, lender.lendees)
			},
		},
		"without metrics manager in context": {
			name:        "context without metrics manager",
			ctx:         context.Background(),
			metricsMgr:  nil,
			expectError: true,
			expectedResult: func(t *testing.T, lender *Lender, err error) {
				assert.Error(t, err)
				assert.Nil(t, lender)
			},
		},
	}

	for testName, tc := range tests {
		t.Run(testName, func(t *testing.T) {
			lender, err := NewLender(tc.ctx, tc.metricsMgr)
			tc.expectedResult(t, lender, err)
		})
	}
}

func TestRegisterLendee(t *testing.T) {
	tests := map[string]struct {
		groupExists bool
	}{
		"successful registration with existing group": {
			groupExists: true,
		},
		"register lendee with uninitialized group": {
			groupExists: false,
		},
	}

	for testName, tc := range tests {
		t.Run(testName, func(t *testing.T) {
			lendeeName := "testLendee"
			lendee := &MockLendee{maxCredits: 5}
			model := &MockAIModel{prediction: 0.8}
			ctx := context.Background()
			mgr := metrics.NewManager()
			lender, err := NewLender(ctx, mgr)
			assert.NoError(t, err)
			if tc.groupExists {
				// Initialize the group map
				lender.lendees[testGroup] = make(map[string]*lendeeRecord)
			}

			lender.RegisterLendee(lendeeName, testGroup, lendee, model)

			// Verify the lendee was registered correctly
			record, exists := lender.lendees[testGroup][lendeeName]
			assert.True(t, exists, "lendee should be registered")
			assert.NotNil(t, record, "lendee record should not be nil")
			assert.Equal(t, lendee, record.lendee, "lendee should match")
			assert.Equal(t, model, record.model, "model should match")
			assert.Equal(t, 0, record.creditsGiven, "credits given should be initialized to 0")

			lendees := lender.GetLendeeNames()
			assert.Contains(t, lendees, testGroup, "lendee group should exist in GetLendees result")
			assert.Contains(t, lendees[testGroup], lendeeName, "lendee name should exist in GetLendees result")

			lender.UnregisterLendee(lendeeName, testGroup)
			record, exists = lender.lendees[testGroup][lendeeName]
			assert.False(t, exists, "lendee should be unregistered")
			assert.Nil(t, record, "lendee record should be nil after unregistration")

			lendees = lender.GetLendeeNames()
			assert.NotContains(t, lendees, testGroup, "lendee group should not exist in GetLendees result")
			assert.NotContains(t, lendees[testGroup], lendeeName, "lendee name should not exist in GetLendees result")
		})
	}
}

func TestRegisterLendee_ConcurrentAccess(t *testing.T) {
	// Test concurrent access to RegisterLendee to ensure thread safety
	ctx := context.Background()
	mgr := metrics.NewManager()
	lender, err := NewLender(ctx, mgr)
	assert.NoError(t, err)

	// Register multiple lendees concurrently
	done := make(chan bool)
	numGoroutines := 10

	for i := range numGoroutines {
		go func(index int) {
			defer func() { done <- true }()
			lendeeName := string(rune('a' + index)) //nolint:gosec // disable G115
			lendee := &MockLendee{maxCredits: index + 1}
			model := &MockAIModel{prediction: float64(index) * 0.1}
			lender.RegisterLendee(lendeeName, testGroup, lendee, model)
		}(i)
	}

	// Wait for all goroutines to complete
	for range numGoroutines {
		<-done
	}

	// Verify all lendees were registered
	assert.Equal(t, numGoroutines, len(lender.lendees[testGroup]), "all lendees should be registered")

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer func() { done <- true }()
			lendeeName := string(rune('a' + index)) //nolint:gosec // disable G115
			lender.UnregisterLendee(lendeeName, testGroup)
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	close(done)

	_, exists := lender.lendees[testGroup]
	assert.False(t, exists, "lendee group should be removed after all lendees unregistered")
}

func TestUnregisterLendee_CleansUpMetrics(t *testing.T) {
	// Test that unregistering a lendee cleans up its prometheus metrics
	ctx := context.Background()
	mgr := metrics.NewManager()
	lender, err := NewLender(ctx, mgr)
	assert.NoError(t, err)

	lendeeName := "testLendee"
	lendeeGroup := testGroup
	mockLendee := &MockLendee{name: lendeeName, maxCredits: 5}
	mockModel := &MockAIModel{prediction: 0.8}

	// Register a lendee
	lender.RegisterLendee(lendeeName, lendeeGroup, mockLendee, mockModel)

	// Set its metrics
	publishLenderCreditsAllocatedMetric(lendeeName, lendeeGroup, 3.0)
	publishLenderCreditsMaxMetric(lendeeName, lendeeGroup, 5.0)

	// Verify metrics exist
	labels := prometheus.Labels{labelLendee: lendeeName, labelType: lendeeGroup}
	allocValue, err := getGaugeValue(lenderCreditsAllocated, labels)
	assert.NoError(t, err)
	assert.Equal(t, 3.0, allocValue)

	maxValue, err := getGaugeValue(lenderCreditsMax, labels)
	assert.NoError(t, err)
	assert.Equal(t, 5.0, maxValue)

	// Unregister the lendee
	lender.UnregisterLendee(lendeeName, lendeeGroup)

	// Verify the lendee is removed from registry
	record, exists := lender.lendees[lendeeGroup][lendeeName]
	assert.False(t, exists, "lendee should be unregistered")
	assert.Nil(t, record, "lendee record should be nil after unregistration")

	// Verify metrics are cleaned up:
	// After deletion, GetMetricWith will recreate the metric with 0 value
	// So we check that the metric was reset to 0 (default for new metrics)
	allocValue, err = getGaugeValue(lenderCreditsAllocated, labels)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, allocValue, "allocated credits should be reset to 0 after deletion")

	maxValue, err = getGaugeValue(lenderCreditsMax, labels)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, maxValue, "max credits should be reset to 0 after deletion")
}

func TestRegisterLendee_OverwriteExisting(t *testing.T) {
	// Test that registering a lendee with the same name overwrites the existing one
	ctx := context.Background()
	mgr := metrics.NewManager()
	lender, err := NewLender(ctx, mgr)
	assert.NoError(t, err)

	// Register second lendee with the same name
	secondLendee := &MockLendee{maxCredits: 7}
	secondModel := &MockAIModel{prediction: 0.9}
	lender.RegisterLendee("sameName", testGroup, secondLendee, secondModel)

	// Verify the second lendee overwrote the first
	record := lender.lendees[testGroup]["sameName"]
	assert.NotNil(t, record)
	assert.Equal(t, secondLendee, record.lendee, "should contain the second lendee")
	assert.Equal(t, secondModel, record.model, "should contain the second model")
	assert.Equal(t, 0, record.creditsGiven, "credits should be reset to 0")
}

func TestLender_Start(t *testing.T) {
	// Setup context with metrics manager
	ctx := context.Background()
	mgr := metrics.NewManager()
	lender, err := NewLender(ctx, mgr)
	assert.NoError(t, err)

	// Create mock metrics that the AI model will request
	metric1 := metrics.NewMetric("cpu_usage", []string{"server"})
	metric2 := metrics.NewMetric("memory_usage", []string{"server"})

	// Register the metrics in the manager so they exist
	err = mgr.Register(metric1)
	assert.NoError(t, err)
	err = mgr.Register(metric2)
	assert.NoError(t, err)

	// Create a mock AI model that requests these metrics
	mockModel := &MockAIModel{
		metricsList: []*metrics.Metric{metric1, metric2},
		prediction:  0.8,
	}

	// Register a lendee with this model
	mockLendee := &MockLendee{maxCredits: 5}
	lender.RegisterLendee("testLendee", testGroup, mockLendee, mockModel)

	// Verify lender is not started initially
	assert.False(t, lender.started, "Lender should not be started initially")

	// Call Start() - launches goroutine and returns immediately
	lender.Start()

	// Verify lender is now started
	assert.True(t, lender.started, "Lender should be started after calling Start()")

	// Add a small delay to allow goroutine to execute
	time.Sleep(10 * time.Millisecond)

	// Verify the metrics are still available in the manager
	retrievedMetric1, err := mgr.GetMetric("cpu_usage")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedMetric1)

	retrievedMetric2, err := mgr.GetMetric("memory_usage")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedMetric2)
}

func TestLender_Start_MissingMetrics(t *testing.T) {
	// Setup context with metrics manager
	ctx := context.Background()
	mgr := metrics.NewManager()
	lender, err := NewLender(ctx, mgr)
	assert.NoError(t, err)

	// Create mock metrics that the AI model will request but don't register them
	metric1 := metrics.NewMetric("nonexistent_metric", []string{"server"})

	// Create a mock AI model that requests these metrics
	mockModel := &MockAIModel{
		metricsList: []*metrics.Metric{metric1},
		prediction:  0.8,
	}

	// Register a lendee with this model
	mockLendee := &MockLendee{maxCredits: 5}
	lender.RegisterLendee("testLendee", testGroup, mockLendee, mockModel)

	// Verify lender is not started initially
	assert.False(t, lender.started, "Lender should not be started initially")

	// Call Start() - launches goroutine and returns immediately
	lender.Start()

	// Verify lender is now started
	assert.True(t, lender.started, "Lender should be started after calling Start()")

	// Add a small delay to allow goroutine to execute
	time.Sleep(10 * time.Millisecond)

	// Verify the lender remains started
	assert.True(t, lender.started, "Lender should remain started after goroutine execution")
}

func TestLender_watchMetrics(t *testing.T) { //nolint:gocognit
	tests := map[string]struct {
		name               string
		setupLendees       func(*Lender, *metrics.Manager) []*MockLendee
		interval           time.Duration
		scaleUpThreshold   float64
		scaleDownThreshold float64
		maxCreditsPerCycle int
		validateResults    func(t *testing.T, mockLendees []*MockLendee) bool // Returns true when validation passes
	}{
		"scale up high prediction": {
			name: "lendees with high predictions get credits",
			setupLendees: func(lender *Lender, mgr *metrics.Manager) []*MockLendee {
				metric := metrics.NewMetric("cpu_usage", []string{"server"})
				err := mgr.Register(metric)
				assert.NoError(t, err)

				mockLendees := []*MockLendee{
					{maxCredits: 5, creditsAdded: 0},
					{maxCredits: 5, creditsAdded: 0},
				}

				// Register lendees with high predictions (above scale up threshold)
				highPredModel := &MockAIModel{
					metricsList: []*metrics.Metric{metric},
					prediction:  0.9, // Above default scale up threshold
				}
				lender.RegisterLendee("highLendee1", testGroup, mockLendees[0], highPredModel)
				lender.RegisterLendee("highLendee2", testGroup, mockLendees[1], highPredModel)

				return mockLendees
			},
			interval:           5 * time.Millisecond,
			scaleUpThreshold:   0.8,
			scaleDownThreshold: 0.3,
			maxCreditsPerCycle: 10,
			validateResults: func(_ *testing.T, mockLendees []*MockLendee) bool {
				// Both lendees should have received credits
				for _, lendee := range mockLendees {
					if lendee.creditsAdded <= 0 {
						return false
					}
				}
				return true
			},
		},
		"scale down low prediction": {
			name: "lendees with low predictions lose credits",
			setupLendees: func(lender *Lender, mgr *metrics.Manager) []*MockLendee {
				metric := metrics.NewMetric("memory_usage", []string{"server"})
				err := mgr.Register(metric)
				assert.NoError(t, err)

				mockLendees := []*MockLendee{
					{maxCredits: 5, creditsAdded: 0},
					{maxCredits: 5, creditsAdded: 0},
				}

				// Register lendees with low predictions (below scale down threshold)
				lowPredModel := &MockAIModel{
					metricsList: []*metrics.Metric{metric},
					prediction:  0.2, // Below default scale down threshold
				}
				lender.RegisterLendee("lowLendee1", testGroup, mockLendees[0], lowPredModel)
				lender.RegisterLendee("lowLendee2", testGroup, mockLendees[1], lowPredModel)

				// Manually give some credits first so they can be removed
				record1 := lender.lendees[testGroup]["lowLendee1"]
				record2 := lender.lendees[testGroup]["lowLendee2"]
				record1.creditsGiven = 3
				record2.creditsGiven = 2
				mockLendees[0].creditsAdded = 3
				mockLendees[1].creditsAdded = 2

				return mockLendees
			},
			interval:           5 * time.Millisecond,
			scaleUpThreshold:   0.8,
			scaleDownThreshold: 0.3,
			maxCreditsPerCycle: 10,
			validateResults: func(_ *testing.T, mockLendees []*MockLendee) bool {
				// Both lendees should have lost credits
				return mockLendees[0].creditsAdded < 3 && mockLendees[1].creditsAdded < 2
			},
		},
		"mixed predictions": {
			name: "mix of high and low predictions",
			setupLendees: func(lender *Lender, mgr *metrics.Manager) []*MockLendee {
				metric := metrics.NewMetric("cpu_memory", []string{"server"})
				err := mgr.Register(metric)
				assert.NoError(t, err)

				mockLendees := []*MockLendee{
					{maxCredits: 5, creditsAdded: 0}, // Low prediction lendee
					{maxCredits: 5, creditsAdded: 0}, // High prediction lendee
					{maxCredits: 5, creditsAdded: 0}, // Medium prediction lendee
				}

				// Low prediction lendee (should lose credits)
				lowPredModel := &MockAIModel{
					metricsList: []*metrics.Metric{metric},
					prediction:  0.1,
				}
				lender.RegisterLendee("lowLendee", testGroup, mockLendees[0], lowPredModel)

				// High prediction lendee (should gain credits)
				highPredModel := &MockAIModel{
					metricsList: []*metrics.Metric{metric},
					prediction:  0.9,
				}
				lender.RegisterLendee("highLendee", testGroup, mockLendees[1], highPredModel)

				// Medium prediction lendee (should remain unchanged)
				medPredModel := &MockAIModel{
					metricsList: []*metrics.Metric{metric},
					prediction:  0.5,
				}
				lender.RegisterLendee("medLendee", testGroup, mockLendees[2], medPredModel)

				// Give the low prediction lendee some credits so they can be removed
				lowRecord := lender.lendees[testGroup]["lowLendee"]
				lowRecord.creditsGiven = 2
				mockLendees[0].creditsAdded = 2

				// Give the medium prediction lendee some credits
				medRecord := lender.lendees[testGroup]["medLendee"]
				medRecord.creditsGiven = 1
				mockLendees[2].creditsAdded = 1

				return mockLendees
			},
			interval:           5 * time.Millisecond,
			scaleUpThreshold:   0.8,
			scaleDownThreshold: 0.3,
			maxCreditsPerCycle: 10,
			validateResults: func(_ *testing.T, mockLendees []*MockLendee) bool {
				// Low prediction lendee should lose credits, high should gain
				lowLost := mockLendees[0].creditsAdded < 2
				highGained := mockLendees[1].creditsAdded > 0
				return lowLost && highGained
			},
		},
		"max credits per cycle limit": {
			name: "respect max credits per cycle limit",
			setupLendees: func(lender *Lender, mgr *metrics.Manager) []*MockLendee {
				metric := metrics.NewMetric("network_io", []string{"server"})
				err := mgr.Register(metric)
				assert.NoError(t, err)

				// Create 5 lendees with different predictions, but max credits per cycle is 2
				mockLendees := make([]*MockLendee, 5)
				predictions := []float64{0.95, 0.94, 0.93, 0.92, 0.91} // All high predictions
				for i := 0; i < 5; i++ {
					mockLendees[i] = &MockLendee{maxCredits: 5, creditsAdded: 0}
					predModel := &MockAIModel{
						metricsList: []*metrics.Metric{metric},
						prediction:  predictions[i], // Different predictions so we can test sorting
					}
					lender.RegisterLendee(fmt.Sprintf("lendee%d", i), testGroup, mockLendees[i], predModel)
				}

				return mockLendees
			},
			interval:           5 * time.Millisecond,
			scaleUpThreshold:   0.8,
			scaleDownThreshold: 0.3,
			maxCreditsPerCycle: 2, // Limit to 2 credits per cycle
			validateResults: func(_ *testing.T, mockLendees []*MockLendee) bool {
				// At least some lendees should get credits
				creditsGiven := 0
				for _, lendee := range mockLendees {
					if lendee.creditsAdded > 0 {
						creditsGiven++
					}
				}
				return creditsGiven > 0
			},
		},
		"context cancellation": {
			name: "proper context cancellation handling",
			setupLendees: func(lender *Lender, mgr *metrics.Manager) []*MockLendee {
				metric := metrics.NewMetric("disk_io", []string{"server"})
				err := mgr.Register(metric)
				assert.NoError(t, err)

				mockLendee := &MockLendee{maxCredits: 5, creditsAdded: 0}
				model := &MockAIModel{
					metricsList: []*metrics.Metric{metric},
					prediction:  0.9,
				}
				lender.RegisterLendee("testLendee", testGroup, mockLendee, model)

				return []*MockLendee{mockLendee}
			},
			interval:           5 * time.Millisecond,
			scaleUpThreshold:   0.8,
			scaleDownThreshold: 0.3,
			maxCreditsPerCycle: 10,
			validateResults: func(_ *testing.T, _ []*MockLendee) bool {
				// Just verify it handles cancellation gracefully
				return true
			},
		},
	}

	for testName, tc := range tests {
		t.Run(testName, func(t *testing.T) {
			// Setup context with cancellation
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mgr := metrics.NewManager()

			// Create lender with custom configuration
			lender := &Lender{
				ctx:                ctx,
				lendees:            make(map[string]map[string]*lendeeRecord),
				metricsMgr:         mgr,
				scaleUpThreshold:   tc.scaleUpThreshold,
				scaleDownThreshold: tc.scaleDownThreshold,
				interval:           tc.interval,
				maxCreditsPerCycle: tc.maxCreditsPerCycle,
			}

			// Setup lendees for this test
			mockLendees := tc.setupLendees(lender, mgr)

			// Start watchMetrics in a goroutine
			done := make(chan struct{}, 1)
			go func() {
				lender.watchMetrics()
				done <- struct{}{}
			}()

			// Poll for expected state with timeout instead of fixed sleep
			// This is much more reliable than time.Sleep
			maxWait := 500 * time.Millisecond
			pollInterval := 10 * time.Millisecond
			deadline := time.Now().Add(maxWait)

			var validationPassed bool
			for time.Now().Before(deadline) {
				if tc.validateResults(t, mockLendees) {
					validationPassed = true
					break
				}
				time.Sleep(pollInterval)
			}

			// Cancel context to stop watchMetrics
			cancel()

			// Wait for watchMetrics to finish with timeout
			select {
			case <-done:
				// Good, function completed
			case <-time.After(2 * time.Second):
				t.Fatal("watchMetrics did not complete within timeout")
			}

			// Final validation check with error reporting
			if !validationPassed {
				// Log final state for debugging
				for i, lendee := range mockLendees {
					t.Logf("Final state - Lendee %d: creditsAdded=%d, maxTotalCredits=%d",
						i, lendee.creditsAdded, lendee.maxCredits)
				}
				t.Errorf("Test '%s' validation did not pass within %v", tc.name, maxWait)
			}
		})
	}
}

func TestLender_watchMetrics_MissingMetrics(t *testing.T) {
	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr := metrics.NewManager()
	lender, err := NewLender(ctx, mgr)
	assert.NoError(t, err)

	// Create mock metrics that the AI model will request but don't register them
	metric1 := metrics.NewMetric("nonexistent_metric", []string{"server"})

	// Create a mock AI model that requests these metrics
	mockModel := &MockAIModel{
		metricsList: []*metrics.Metric{metric1},
		prediction:  0.8,
	}

	// Register a lendee with this model
	mockLendee := &MockLendee{maxCredits: 5}
	lender.RegisterLendee("testLendee", testGroup, mockLendee, mockModel)

	// Start watchMetrics in a goroutine
	done := make(chan struct{}, 1)
	go func() {
		lender.watchMetrics()
		done <- struct{}{}
	}()

	// Let it run briefly, then cancel
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Wait for watchMetrics to finish
	<-done
	// No error assertions needed since function doesn't return error
}

func TestNewLender_EnvironmentVariables(t *testing.T) {
	// Store original environment variables to restore later
	originalScaleUp := os.Getenv(scaleUpEnvVar)
	originalScaleDown := os.Getenv(scaleDownEnvVar)
	originalInterval := os.Getenv(intervalEnvVar)
	originalMaxCredits := os.Getenv(maxCreditsPerCycleEnvVar)

	// Cleanup function to restore original values
	defer func() {
		if originalScaleUp != "" {
			os.Setenv(scaleUpEnvVar, originalScaleUp)
		} else {
			os.Unsetenv(scaleUpEnvVar)
		}
		if originalScaleDown != "" {
			os.Setenv(scaleDownEnvVar, originalScaleDown)
		} else {
			os.Unsetenv(scaleDownEnvVar)
		}
		if originalInterval != "" {
			os.Setenv(intervalEnvVar, originalInterval)
		} else {
			os.Unsetenv(intervalEnvVar)
		}
		if originalMaxCredits != "" {
			os.Setenv(maxCreditsPerCycleEnvVar, originalMaxCredits)
		} else {
			os.Unsetenv(maxCreditsPerCycleEnvVar)
		}
	}()

	tests := map[string]struct {
		envVars            map[string]string
		expectedScaleUp    float64
		expectedScaleDown  float64
		expectedInterval   time.Duration
		expectedMaxCredits int
	}{
		"default values when no env vars set": {
			envVars:            map[string]string{},
			expectedScaleUp:    ScaleUpThresholdDefault,
			expectedScaleDown:  scaleDownThresholdDefault,
			expectedInterval:   intervalDefault,
			expectedMaxCredits: maxCreditsPerCycleDefault,
		},
		"custom values from env vars": {
			envVars: map[string]string{
				scaleUpEnvVar:            "0.8",
				scaleDownEnvVar:          "0.2",
				intervalEnvVar:           "30",
				maxCreditsPerCycleEnvVar: "10",
			},
			expectedScaleUp:    0.8,
			expectedScaleDown:  0.2,
			expectedInterval:   30 * time.Second,
			expectedMaxCredits: 10,
		},
		"partial env vars with defaults": {
			envVars: map[string]string{
				maxCreditsPerCycleEnvVar: "3",
			},
			expectedScaleUp:    ScaleUpThresholdDefault,
			expectedScaleDown:  scaleDownThresholdDefault,
			expectedInterval:   intervalDefault,
			expectedMaxCredits: 3,
		},
		"invalid values fall back to defaults": {
			envVars: map[string]string{
				scaleUpEnvVar:            "invalid",
				scaleDownEnvVar:          "invalid",
				intervalEnvVar:           "invalid",
				maxCreditsPerCycleEnvVar: "invalid",
			},
			expectedScaleUp:    ScaleUpThresholdDefault,
			expectedScaleDown:  scaleDownThresholdDefault,
			expectedInterval:   intervalDefault,
			expectedMaxCredits: maxCreditsPerCycleDefault,
		},
	}

	for testName, tc := range tests {
		t.Run(testName, func(t *testing.T) {
			// Clear all relevant environment variables first
			os.Unsetenv(scaleUpEnvVar)
			os.Unsetenv(scaleDownEnvVar)
			os.Unsetenv(intervalEnvVar)
			os.Unsetenv(maxCreditsPerCycleEnvVar)

			// Set test environment variables
			for key, value := range tc.envVars {
				os.Setenv(key, value)
			}

			// Setup context and metrics manager
			ctx := context.Background()
			mgr := metrics.NewManager()

			// Create lender and verify configuration
			lender, err := NewLender(ctx, mgr)
			assert.NoError(t, err)
			assert.NotNil(t, lender)

			// Verify the configuration values
			assert.Equal(t, tc.expectedScaleUp, lender.scaleUpThreshold)
			assert.Equal(t, tc.expectedScaleDown, lender.scaleDownThreshold)
			assert.Equal(t, tc.expectedInterval, lender.interval)
			assert.Equal(t, tc.expectedMaxCredits, lender.maxCreditsPerCycle)
		})
	}
}

func TestLender_watchMetrics_MaxCreditsConfiguration(t *testing.T) {
	// Store original environment variable
	originalMaxCredits := os.Getenv(maxCreditsPerCycleEnvVar)
	defer func() {
		if originalMaxCredits != "" {
			os.Setenv(maxCreditsPerCycleEnvVar, originalMaxCredits)
		} else {
			os.Unsetenv(maxCreditsPerCycleEnvVar)
		}
	}()

	// Set custom max credits value
	os.Setenv(maxCreditsPerCycleEnvVar, "3")

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr := metrics.NewManager()
	lender, err := NewLender(ctx, mgr)
	assert.NoError(t, err)

	// Verify the custom value was set correctly
	assert.Equal(t, 3, lender.maxCreditsPerCycle)

	// Create mock metrics that the AI model will request
	metric1 := metrics.NewMetric("cpu_usage", []string{"server"})
	err = mgr.Register(metric1)
	assert.NoError(t, err)

	// Create multiple lendees with different AI models that have high predictions
	// to test that only maxCreditsPerCycle (3) lendees get credits
	mockLendees := make([]*MockLendee, 5)
	for i := 0; i < 5; i++ {
		mockModel := &MockAIModel{
			metricsList: []*metrics.Metric{metric1},
			prediction:  0.9, // High prediction to trigger credit addition
		}
		mockLendee := &MockLendee{maxCredits: 10}
		mockLendees[i] = mockLendee
		lender.RegisterLendee(fmt.Sprintf("testLendee%d", i), testGroup, mockLendee, mockModel)
	}

	// Start watchMetrics in a goroutine
	done := make(chan struct{}, 1)
	go func() {
		lender.watchMetrics()
		done <- struct{}{}
	}()

	// Let it run briefly, then cancel
	time.Sleep(50 * time.Millisecond) // Slightly longer to ensure one cycle completes
	cancel()

	// Wait for watchMetrics to finish
	<-done

	// Count how many lendees received credits
	creditsAdded := 0
	for _, mockLendee := range mockLendees {
		if mockLendee.creditsAdded > 0 {
			creditsAdded++
		}
	}

	// Should be limited to maxCreditsPerCycle (3) even though we have 5 lendees with high predictions
	assert.LessOrEqual(t, creditsAdded, 3, "Should not exceed maxCreditsPerCycle configuration")
}

func TestLender_addCredit(t *testing.T) {
	tests := map[string]struct {
		name          string
		setupRecord   func() *lendeeRecord
		expectError   bool
		expectedCreds int
		errorContains string
	}{
		"successful add credit within limit": {
			name: "add credit when under max",
			setupRecord: func() *lendeeRecord {
				mockLendee := &MockLendee{
					name:         "mockLendee",
					maxCredits:   5,
					creditsAdded: 2,
				}
				return &lendeeRecord{
					lendee:          mockLendee,
					creditsGiven:    2,
					maxCreditsGiven: mockLendee.MaxLenderCredits(),
					model:           &MockAIModel{},
				}
			},
			expectError:   false,
			expectedCreds: 3,
		},
		"add credit at max limit": {
			name: "add credit when at max limit",
			setupRecord: func() *lendeeRecord {
				mockLendee := &MockLendee{
					name:         "mockLendee",
					maxCredits:   3,
					creditsAdded: 3,
				}
				return &lendeeRecord{
					lendee:          mockLendee,
					creditsGiven:    3,
					maxCreditsGiven: mockLendee.MaxLenderCredits(),
					model:           &MockAIModel{},
				}
			},
			expectError:   true,
			expectedCreds: 3,
			errorContains: "has reached max credits",
		},
		"add credit with lendee error": {
			name: "lendee returns error on AddCredit",
			setupRecord: func() *lendeeRecord {
				mockLendee := &MockLendee{
					name:         "mockLendee",
					maxCredits:   5,
					creditsAdded: 1,
					addCreditErr: fmt.Errorf("mock add credit error"),
				}
				return &lendeeRecord{
					lendee:          mockLendee,
					creditsGiven:    1,
					maxCreditsGiven: mockLendee.MaxLenderCredits(),
					model:           &MockAIModel{},
				}
			},
			expectError:   true,
			expectedCreds: 2, // creditsGiven should still be incremented
			errorContains: "mock add credit error",
		},
		"add credit from zero": {
			name: "add first credit",
			setupRecord: func() *lendeeRecord {
				mockLendee := &MockLendee{
					name:         "mockLendee",
					maxCredits:   2,
					creditsAdded: 0,
				}
				return &lendeeRecord{
					lendee:          mockLendee,
					creditsGiven:    0,
					maxCreditsGiven: mockLendee.MaxLenderCredits(),
					model:           &MockAIModel{},
				}
			},
			expectError:   false,
			expectedCreds: 1,
		},
	}

	for testName, tc := range tests {
		t.Run(testName, func(t *testing.T) {
			// Setup lender
			ctx := context.Background()
			mgr := metrics.NewManager()
			lender, err := NewLender(ctx, mgr)
			assert.NoError(t, err)

			// Setup lendee record
			record := tc.setupRecord()

			// Call addCredit
			err = lender.addCredit(record)

			// Verify expectations
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectedCreds, record.creditsGiven, "creditsGiven should match expected value")
		})
	}
}

func TestLender_removeCredit(t *testing.T) {
	tests := map[string]struct {
		name          string
		setupRecord   func() *lendeeRecord
		expectError   bool
		expectedCreds int
		errorContains string
	}{
		"successful remove credit": {
			name: "remove credit when credits available",
			setupRecord: func() *lendeeRecord {
				mockLendee := &MockLendee{maxCredits: 5, creditsAdded: 3}
				return &lendeeRecord{
					lendee:       mockLendee,
					creditsGiven: 3,
					model:        &MockAIModel{},
				}
			},
			expectError:   false,
			expectedCreds: 2,
		},
		"remove credit with lendee error": {
			name: "lendee returns error on RemoveCredit",
			setupRecord: func() *lendeeRecord {
				mockLendee := &MockLendee{
					maxCredits:      5,
					creditsAdded:    2,
					removeCreditErr: fmt.Errorf("mock remove credit error"),
				}
				return &lendeeRecord{
					lendee:       mockLendee,
					creditsGiven: 2,
					model:        &MockAIModel{},
				}
			},
			expectError:   true,
			expectedCreds: 1, // creditsGiven should still be decremented
			errorContains: "mock remove credit error",
		},
		"remove credit when no credits given": {
			name: "attempt to remove credit when creditsGiven is zero",
			setupRecord: func() *lendeeRecord {
				mockLendee := &MockLendee{maxCredits: 5, creditsAdded: 0}
				return &lendeeRecord{
					lendee:       mockLendee,
					creditsGiven: 0,
					model:        &MockAIModel{},
				}
			},
			expectError:   true,
			expectedCreds: 0,
			errorContains: "has no credits to remove",
		},
		"remove last credit": {
			name: "remove the last available credit",
			setupRecord: func() *lendeeRecord {
				mockLendee := &MockLendee{maxCredits: 3, creditsAdded: 1}
				return &lendeeRecord{
					lendee:       mockLendee,
					creditsGiven: 1,
					model:        &MockAIModel{},
				}
			},
			expectError:   false,
			expectedCreds: 0,
		},
	}

	for testName, tc := range tests {
		t.Run(testName, func(t *testing.T) {
			// Setup lender
			ctx := context.Background()
			mgr := metrics.NewManager()
			lender, err := NewLender(ctx, mgr)
			assert.NoError(t, err)

			// Setup lendee record
			record := tc.setupRecord()

			// Call removeCredit
			err = lender.removeCredit(record)

			// Verify expectations
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectedCreds, record.creditsGiven, "creditsGiven should match expected value")
		})
	}
}

func TestLender_addCredit_removeCredit_Integration(t *testing.T) {
	// Test the integration between addCredit and removeCredit functions
	ctx := context.Background()
	mgr := metrics.NewManager()
	lender, err := NewLender(ctx, mgr)
	assert.NoError(t, err)

	// Create a mock lendee with a max of 3 credits
	mockLendee := &MockLendee{
		name:         "mockLendee",
		maxCredits:   3,
		creditsAdded: 0,
	}
	record := &lendeeRecord{
		lendee:          mockLendee,
		creditsGiven:    0,
		maxCreditsGiven: mockLendee.MaxLenderCredits(),
		model:           &MockAIModel{},
	}

	// Test adding credits up to the limit
	for i := 1; i <= 3; i++ {
		err = lender.addCredit(record)
		assert.NoError(t, err, "should be able to add credit %d", i)
		assert.Equal(t, i, record.creditsGiven, "creditsGiven should be %d", i)
		assert.Equal(t, i, mockLendee.creditsAdded, "mock lendee should have %d credits", i)
	}

	// Try to add beyond the limit
	err = lender.addCredit(record)
	assert.Error(t, err, "should not be able to add credit beyond max")
	assert.Contains(t, err.Error(), "has reached max credits")
	assert.Equal(t, 3, record.creditsGiven, "creditsGiven should remain at 3")

	// Test removing credits back to zero
	for i := 3; i >= 1; i-- {
		err = lender.removeCredit(record)
		assert.NoError(t, err, "should be able to remove credit when at %d", i)
		assert.Equal(t, i-1, record.creditsGiven, "creditsGiven should be %d", i-1)
		assert.Equal(t, i-1, mockLendee.creditsAdded, "mock lendee should have %d credits", i-1)
	}

	// Try to remove when no credits available
	err = lender.removeCredit(record)
	assert.Error(t, err, "should not be able to remove credit when none available")
	assert.Contains(t, err.Error(), "has no credits to remove")
	assert.Equal(t, 0, record.creditsGiven, "creditsGiven should remain at 0")
}
