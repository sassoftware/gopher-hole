package metrics

import (
	"fmt"
	"time"

	"github.com/sassoftware/gopher-hole/util/math"
	"github.com/sassoftware/gopher-hole/util/safebuffer"
)

var (
	// Make this func able to be overridden for unit testing purposes
	nowFunc = time.Now
)

const (
	recordCapacity                = 50
	minRecordsForTrendCalculation = 5
)

type Metric struct {
	name    string
	records *safebuffer.SafeBuffer
	labels  []string
}

type record struct {
	value     float64
	timestamp time.Time
}

// NewMetric initializes a new Metric struct.
func NewMetric(name string, labels []string) *Metric {
	return &Metric{
		name:    name,
		records: safebuffer.New(recordCapacity),
		labels:  labels,
	}
}

// insertRecord records and stores a value at the given time.
func (m *Metric) insertRecord(value float64, timestamp time.Time) {
	m.records.Add(record{
		value:     value,
		timestamp: timestamp,
	})
}

// InsertRecord records and stores a value at the current time.
func (m *Metric) InsertRecord(value float64) {
	m.insertRecord(value, time.Now())
}

// GetName returns the name of the metric.
func (m *Metric) GetName() string {
	return m.name
}

func (m *Metric) getRecords() []record {
	data := m.records.GetData()
	var recordsSlice []record
	for _, entry := range data {
		if r, ok := entry.(record); ok {
			recordsSlice = append(recordsSlice, r)
		}
	}
	return recordsSlice
}

func (m *Metric) getRecordsWithinTimeframe(duration time.Duration) []record {
	records := m.getRecords()
	endTime := nowFunc()
	startTime := endTime.Add(-duration)

	var filteredRecords []record
	for _, r := range records {
		if r.timestamp.After(startTime) {
			filteredRecords = append(filteredRecords, r)
		}
	}
	return filteredRecords
}

// GetMostRecent returns the value of the metric's most recent record within the specified timeframe.
// If no records exist within the timeframe, it returns 0 and an error.
func (m *Metric) GetMostRecent(duration time.Duration) (float64, error) {
	records := m.getRecordsWithinTimeframe(duration)
	if len(records) == 0 {
		return 0, fmt.Errorf("no records found within the given timeframe")
	}
	// Find the most recent record
	mostRecent := records[0]
	for _, r := range records {
		if r.timestamp.After(mostRecent.timestamp) {
			mostRecent = r
		}
	}
	return mostRecent.value, nil
}

// Trend calculates and returns the linear regression slope of the records
// that fall within the given duration.
// The number returned represents units/second.
func (m *Metric) Trend(duration time.Duration) (float64, error) {
	records := m.getRecordsWithinTimeframe(duration)
	filteredDataPoints := convertRecordsToDataPoints(records)
	if len(filteredDataPoints) < minRecordsForTrendCalculation {
		return 0.0, fmt.Errorf("not enough records found within the given timeframe to accurately calculate trend")
	}
	return math.CalculateLinearRegressionSlope(filteredDataPoints)
}

func convertRecordsToDataPoints(records []record) []math.DataPoint {
	var filteredDataPoints []math.DataPoint
	for _, r := range records {
		secondsSinceFirstRecord := r.timestamp.Sub(records[0].timestamp).Seconds()
		filteredDataPoints = append(filteredDataPoints, math.DataPoint{
			X: secondsSinceFirstRecord,
			Y: r.value,
		})
	}
	return filteredDataPoints
}
