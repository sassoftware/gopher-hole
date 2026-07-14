package lender

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/sassoftware/gopher-hole/internal/log"
	"github.com/sassoftware/gopher-hole/key"
	"github.com/sassoftware/gopher-hole/metrics"
)

var (
	// LenderKey is the plugin key for the lender.
	LenderKey = key.Key{Name: "gopher-hole/lender"}
)

const (
	ScaleUpThresholdDefault   = 0.5
	scaleDownThresholdDefault = 0.1
	intervalDefault           = 1 * time.Minute
	maxCreditsPerCycleDefault = 5

	scaleUpEnvVar            = "SAS_EVENT_ELASTIC_SCALE_UP"
	scaleDownEnvVar          = "SAS_EVENT_ELASTIC_SCALE_DOWN"
	intervalEnvVar           = "SAS_EVENT_ELASTIC_INTERVAL"
	maxCreditsPerCycleEnvVar = "SAS_EVENT_ELASTIC_MAX_CREDITS_PER_CYCLE" //nolint:gosec
)

// A CreditBorrower can acquire credits for processing and return them once completed
type CreditBorrower interface {
	// AcquireCredit obtains a processing credit, blocking until one is available or the context is canceled
	AcquireCredit(context.Context) error
	// ReleaseCredit gives the credit back after processing has finished
	ReleaseCredit() error
}

// A Lendee can respond to scaling events by these methods
type Lendee interface {
	// AddCredit increments the Lendee's capacity by 1
	AddCredit() error
	// RemoveCredit decrements the Lendee's capacity by 1
	RemoveCredit(ctx context.Context) error
	// MaxLenderCredits returns the maximum number of credits that can be lent to the Lendee
	MaxLenderCredits() int
	// Availability returns the percentage of credits currently available for use, as a value between 0 and 1
	Availability() float64
	// GetName returns the name of the Lendee
	GetName() string
}

// The lender uses a map of LendeeRecords to keep track of each lendee
type lendeeRecord struct {
	lendee          Lendee
	creditsGiven    int
	maxCreditsGiven int
	model           AIModel
}

// MetricSource is the read-only view the lender needs over a metrics registry:
// the ability to look up a single metric by name. Any type that provides this
// method — such as a metrics.Manager — satisfies it, so the lender never
// depends on a concrete metrics manager implementation.
type MetricSource interface {
	GetMetric(name string) (*metrics.Metric, error)
}

// Lender manages the lending and tracking of resources to multiple lendees.
// It provides thread-safe operations for managing lendee records and includes
// metrics collection capabilities. The Lender uses a nested map structure
// where the outer key represents lendee categories and the inner key represents
// specific lendee identifiers.
type Lender struct {
	ctx                *context.Context
	lendees            map[string]map[string]*lendeeRecord
	lendeesMu          sync.Mutex
	metricSource       MetricSource
	scaleUpThreshold   float64
	scaleDownThreshold float64
	interval           time.Duration
	maxCreditsPerCycle int
	started            bool
}

// NewLender creates a new Lender instance with the provided context and metric source.
// The metric source supplies the metric values the lender evaluates; it is injected
// explicitly rather than pulled from the context so the lender does not depend on a
// concrete metrics manager.
//
// Parameters:
//   - ctx: A pointer to the context used for cancellation
//   - source: The MetricSource the lender queries for metric values
//
// Returns:
//   - *Lender: A pointer to the newly created Lender instance
//   - error: An error if the provided metric source is nil
func NewLender(ctx *context.Context, source MetricSource) (*Lender, error) {
	if source == nil {
		return nil, errors.New("lender: metric source must not be nil")
	}

	return &Lender{
		ctx:                ctx,
		lendees:            make(map[string]map[string]*lendeeRecord),
		metricSource:       source,
		scaleUpThreshold:   getScaleUpThreshold(),
		scaleDownThreshold: getScaleDownThreshold(),
		interval:           getInterval(),
		maxCreditsPerCycle: getMaxCreditsPerCycle(),
	}, nil
}

func getScaleUpThreshold() float64 {
	if scaleUpEnv := os.Getenv(scaleUpEnvVar); scaleUpEnv != "" {
		parsedThreshold, err := strconv.ParseFloat(scaleUpEnv, 64)
		if err == nil {
			log.GetLogger().Debug().Msgf("Scale up threshold configured from environment: %.2f (env=%s)", parsedThreshold, scaleUpEnv)
			return parsedThreshold
		}
		log.GetLogger().Debug().Msgf("Invalid %s value '%s', using default: %v", scaleUpEnvVar, scaleUpEnv, err)
	}
	return ScaleUpThresholdDefault
}

func getScaleDownThreshold() float64 {
	if scaleDownEnv := os.Getenv(scaleDownEnvVar); scaleDownEnv != "" {
		parsedThreshold, err := strconv.ParseFloat(scaleDownEnv, 64)
		if err == nil {
			log.GetLogger().Debug().Msgf("Scale down threshold configured from environment: %.2f (env=%s)", parsedThreshold, scaleDownEnv)
			return parsedThreshold
		}
		log.GetLogger().Debug().Msgf("Invalid %s value '%s', using default: %v", scaleDownEnvVar, scaleDownEnv, err)
	}
	return scaleDownThresholdDefault
}

func getInterval() time.Duration {
	if intervalEnv := os.Getenv(intervalEnvVar); intervalEnv != "" {
		parsedInterval, err := strconv.Atoi(intervalEnv)
		if err == nil {
			interval := time.Duration(parsedInterval) * time.Second
			log.GetLogger().Debug().Msgf("Interval configured from environment: %d seconds (env=%s)", parsedInterval, intervalEnv)
			return interval
		}
		log.GetLogger().Debug().Msgf("Invalid %s value '%s', using default: %v", intervalEnvVar, intervalEnv, err)
	}
	return intervalDefault
}

func getMaxCreditsPerCycle() int {
	if maxCreditsEnv := os.Getenv(maxCreditsPerCycleEnvVar); maxCreditsEnv != "" {
		parsedMaxCredits, err := strconv.Atoi(maxCreditsEnv)
		if err == nil {
			log.GetLogger().Debug().Msgf("Max credits per cycle configured from environment: %d (env=%s)", parsedMaxCredits, maxCreditsEnv)
			return parsedMaxCredits
		}
		log.GetLogger().Debug().Msgf("Invalid %s value '%s', using default: %v", maxCreditsPerCycleEnvVar, maxCreditsEnv, err)
	}
	return maxCreditsPerCycleDefault
}

// RegisterLendee adds a new lendee to the lender's registry under the specified group.
// It creates a new lendeeRecord with the provided lendee instance and AI model,
// initializing the credits given to 0. If the specified lendeeGroup doesn't exist
// in the registry, it creates a new group map. This operation is thread-safe.
//
// Parameters:
//   - lendeeName: unique identifier for the lendee within the group
//   - lendeeGroup: group category to organize lendees
//   - c: the Lendee instance to register
//   - m: the AIModel associated with this lendee
func (l *Lender) RegisterLendee(lendeeName string, lendeeGroup string, c Lendee, m AIModel) {
	log.GetLogger().Debug().Msgf("Registering lendee '%s' in group '%s'", lendeeName, lendeeGroup)
	l.lendeesMu.Lock()
	defer l.lendeesMu.Unlock()

	// Initialize the group map if it doesn't exist
	if l.lendees[lendeeGroup] == nil {
		l.lendees[lendeeGroup] = make(map[string]*lendeeRecord)
	}

	record := &lendeeRecord{
		lendee:          c,
		model:           m,
		creditsGiven:    0,
		maxCreditsGiven: c.MaxLenderCredits(),
	}
	l.lendees[lendeeGroup][lendeeName] = record

	// Initialize prometheus metrics immediately upon registration
	publishLenderCreditsAllocatedMetric(c.GetName(), lendeeGroup, float64(record.creditsGiven))
	publishLenderCreditsMaxMetric(c.GetName(), lendeeGroup, float64(record.maxCreditsGiven))
}

// UnregisterLendee removes a lendee from the lender's registry based on the provided
// lendee name and group. If the specified group becomes empty after removal, it also
// deletes the group from the registry. This operation is thread-safe.
//
// Parameters:
//   - lendeeName: unique identifier for the lendee within the group
//   - lendeeGroup: group category from which to remove the lendee
func (l *Lender) UnregisterLendee(lendeeName string, lendeeGroup string) {
	log.GetLogger().Debug().Msgf("Unregistering lendee '%s' from group '%s'", lendeeName, lendeeGroup)
	l.lendeesMu.Lock()
	defer l.lendeesMu.Unlock()

	if group, exists := l.lendees[lendeeGroup]; exists {
		// Clean up prometheus metrics before removing the lendee
		if record, lendeeExists := group[lendeeName]; lendeeExists {
			removeLenderMetrics(record.lendee.GetName(), lendeeGroup)
		}
		delete(group, lendeeName)
		if len(group) == 0 {
			delete(l.lendees, lendeeGroup)
		}
	}
}

// GetLendeeNames returns a map of the currently registered lendees.
// The key is the group name, and the value is a list of names
// of lendees in that group. This operation is thread-safe.
//
// Returns:
//   - map[string][]string: Names of lendees organized by their group names
func (l *Lender) GetLendeeNames() map[string][]string {
	l.lendeesMu.Lock()
	defer l.lendeesMu.Unlock()

	lendeeNames := make(map[string][]string)
	for groupName, group := range l.lendees {
		lendeeNames[groupName] = make([]string, 0, len(group))
		for lendeeName := range group {
			lendeeNames[groupName] = append(lendeeNames[groupName], lendeeName)
		}
	}
	return lendeeNames
}

// addCredit adds a credit to the lendee and calls its AddCredit function.
func (l *Lender) addCredit(lr *lendeeRecord) error {
	log.GetLogger().Debug().Msgf("Attempting to add credit to lendee '%s': current=%d, max=%d", lr.lendee.GetName(), lr.creditsGiven, lr.maxCreditsGiven)
	if lr.creditsGiven < lr.maxCreditsGiven {
		lr.creditsGiven++
		err := lr.lendee.AddCredit()
		if err != nil {
			logger.Debug().Msgf("Failed to add credit to lendee '%s': %v (creditsGiven=%d)", lr.lendee.GetName(), err, lr.creditsGiven)
		} else {
			logger.Debug().Msgf("Successfully added credit to lendee '%s': creditsGiven=%d", lr.lendee.GetName(), lr.creditsGiven)
		}
		return err
	}
	logger.Debug().Msgf("Cannot add credit: lendee '%s' has reached max credits given (%d)", lr.lendee.GetName(), lr.maxCreditsGiven)
	return fmt.Errorf("lendee '%s' has reached max credits given (%d)", lr.lendee.GetName(), lr.maxCreditsGiven)
}

// removeCredit removes a credit from the lendee and calls its RemoveCredit function.
func (l *Lender) removeCredit(lr *lendeeRecord) error {
	logger.Debug().Msgf("Attempting to remove credit from lendee '%s': current=%d", lr.lendee.GetName(), lr.creditsGiven)
	// Never remove a credit that we didn't issue. Subsystems should initialize
	// with a default credit, we don't want to take any of those away.
	if lr.creditsGiven > 0 {
		lr.creditsGiven--
		err := lr.lendee.RemoveCredit(*l.ctx)
		if err != nil {
			logger.Debug().Msgf("Failed to remove credit from lendee '%s': %v (creditsGiven=%d)", lr.lendee.GetName(), err, lr.creditsGiven)
		} else {
			logger.Debug().Msgf("Successfully removed credit from lendee '%s': creditsGiven=%d", lr.lendee.GetName(), lr.creditsGiven)
		}
		return err
	}
	logger.Debug().Msgf("Cannot remove credit from lendee '%s': no credits given by lender", lr.lendee.GetName())
	return fmt.Errorf("lendee '%s' has no credits to remove", lr.lendee.GetName())
}

// Start launches the metric watching process in a background goroutine.
// This method starts monitoring all registered lendees and their AI models'
// required metrics. It returns immediately after launching the background process.
func (l *Lender) Start() {
	logger.Debug().Msgf("Starting lender metrics monitoring with interval: %v", l.interval)
	l.started = true
	go l.watchMetrics()
}

// watchMetrics continuously monitors all registered lendees, retrieves their AI models' required
// metrics, and requests the current metric values from the metrics subsystem at the configured interval.
// This method is thread-safe and can be interrupted by canceling the lender's context.
func (l *Lender) watchMetrics() { //nolint:gocognit
	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()

	// Wait for ticker before running, then repeat at interval
	for {
		// Wait for tick or context cancellation
		select {
		case <-(*l.ctx).Done():
			return
		case <-ticker.C:
			// Continue to processing
		}

		// Acquire lock briefly to create a copy of lendees
		l.lendeesMu.Lock()
		lendeesCopy := make(map[string]map[string]*lendeeRecord)
		totalLendees := 0
		for groupName, group := range l.lendees {
			lendeesCopy[groupName] = make(map[string]*lendeeRecord)
			maps.Copy(lendeesCopy[groupName], group)
			totalLendees += len(group)
		}
		l.lendeesMu.Unlock()
		logger.Debug().Msgf("Processing %d lendees across %d groups", totalLendees, len(lendeesCopy))

		// Process the copy without holding the lock
		// Collect all predictions with their associated records
		type predictionResult struct {
			record     *lendeeRecord
			prediction float64
		}
		var predictions []predictionResult

		for _, group := range lendeesCopy {
			for _, record := range group {
				// Check for cancellation before processing each record
				select {
				case <-(*l.ctx).Done():
					return
				default:
				}

				// Get the list of metrics this AI model requires
				requiredMetrics := record.model.ListMetrics()
				logger.Debug().Msgf("Lendee '%s' requires %d metrics", record.lendee.GetName(), len(requiredMetrics))

				// Collect successfully retrieved metrics for prediction
				var retrievedMetrics []*metrics.Metric
				for _, metric := range requiredMetrics {
					if metric != nil {
						// Request the metric from the metrics manager
						retrievedMetric, err := l.metricSource.GetMetric(metric.GetName())
						if err != nil {
							logger.Debug().Msgf("Failed to retrieve metric '%s': %v", metric.GetName(), err)
							// Log the error but continue processing other metrics
							// This allows the system to be resilient to missing metrics
							continue
						}
						// Add successfully retrieved metric to the list
						retrievedMetrics = append(retrievedMetrics, retrievedMetric)
					}
				}

				// Execute the AI model's prediction with the retrieved metrics
				if len(retrievedMetrics) > 0 {
					prediction, err := record.model.Predict(retrievedMetrics)
					if err != nil {
						logger.Debug().Msgf("AI model prediction failed for lendee '%s': %v", record.lendee.GetName(), err)
						continue
					}
					logger.Debug().Msgf("AI model prediction for lendee '%s': %.4f (using %d/%d metrics)",
						record.lendee.GetName(),
						prediction,
						len(retrievedMetrics),
						len(requiredMetrics),
					)
					// Collect prediction result with its record
					predictions = append(predictions, predictionResult{
						record:     record,
						prediction: prediction,
					})
				} else {
					logger.Debug().Msgf("Skipping prediction: no metrics successfully retrieved for lendee '%s'", record.lendee.GetName())
				}
			}
		}

		// Sort predictions from lowest to highest
		sort.Slice(predictions, func(i, j int) bool {
			return predictions[i].prediction < predictions[j].prediction
		})

		// Apply credit management logic
		// Remove credits for predictions below scale down threshold
		creditsRemoved := 0
		for _, pred := range predictions {
			if pred.prediction < l.scaleDownThreshold {
				logger.Debug().Msgf("Removing credit from lendee '%s' for prediction %.4f (below threshold %.4f)",
					pred.record.lendee.GetName(),
					pred.prediction,
					l.scaleDownThreshold,
				)
				err := l.removeCredit(pred.record)
				if err == nil { // Only count as removed if we successfully removed a credit
					creditsRemoved++
				}
			} else {
				break // Exit early since predictions are ordered
			}
		}
		logger.Debug().Msgf("Removed %d credits based on low predictions", creditsRemoved)

		// Add credits for the highest predictions, but only if they exceed scale up threshold
		creditsAdded := 0
		if len(predictions) > 0 {
			startIndex := max(len(predictions)-l.maxCreditsPerCycle, 0)
			for i := startIndex; i < len(predictions); i++ {
				if predictions[i].prediction > l.scaleUpThreshold {
					logger.Debug().Msgf("Adding credit to lendee '%s' for prediction %.4f (above threshold %.4f)",
						predictions[i].record.lendee.GetName(),
						predictions[i].prediction,
						l.scaleUpThreshold,
					)
					_ = l.addCredit(predictions[i].record) // Ignore errors for resilience
					creditsAdded++
				}
			}
		}
		logger.Debug().Msgf("Completed metrics evaluation cycle: removed=%d, added=%d credits", creditsRemoved, creditsAdded)

		// Update Prometheus gauges for all lendees
		for groupName, group := range lendeesCopy {
			for _, record := range group {
				name := record.lendee.GetName()
				publishLenderCreditsAllocatedMetric(name, groupName, float64(record.creditsGiven))
				publishLenderCreditsMaxMetric(name, groupName, float64(record.maxCreditsGiven))
			}
		}
	}
}

func (l *Lender) Started() bool {
	return l.started
}
