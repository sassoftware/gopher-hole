package metrics

import (
	"reflect"
	"testing"
	"time"

	"github.com/sassoftware/gopher-hole/util/math"
	"github.com/stretchr/testify/assert"
)

const defaultRecordsSlope = 1.1714285714285715

func Test_Metric_GetName(t *testing.T) {
	testMetric := NewMetric("test", nil)
	assert.Equal(t, "test", testMetric.GetName())
}

func Test_Metric_InsertRecord(t *testing.T) {
	tests := map[string]struct {
		numRecords         int
		numExpectedRecords int
	}{
		"5 records, capacity 50": {
			numRecords:         5,
			numExpectedRecords: 5,
		},
		"55 records, capacity 50": {
			numRecords:         55,
			numExpectedRecords: 50,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			sample := NewMetric("sample", nil)
			value := 10.0
			for i := 0; i < tc.numRecords; i++ {
				sample.InsertRecord(value)
				value += 10.0
				time.Sleep(time.Millisecond)
			}
			recordsSlice := sample.getRecords()
			assert.Equal(t, tc.numExpectedRecords, len(recordsSlice))
			for i := 0; i < tc.numExpectedRecords; i++ {
				assert.Equal(t, float64((i+1+(tc.numRecords-tc.numExpectedRecords))*10), recordsSlice[i].value)
			}
		})
	}
}

func Test_Metric_GetMostRecent(t *testing.T) {
	// Initialize a metric
	testMetric := NewMetric("test", nil)

	// Set now to a fixed time for testing
	nowFunc = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UTC
	defer func() {
		nowFunc = time.Now
	}()

	// Test with no records
	actual, err := testMetric.GetMostRecent(5 * time.Second)
	assert.Error(t, err)
	assert.Equal(t, 0.0, actual)

	// Insert test records
	for _, r := range getDefaultTestRecords() {
		testMetric.insertRecord(r.value, r.timestamp)
	}

	// Test with all records
	actual, err = testMetric.GetMostRecent(7 * time.Second)
	assert.NoError(t, err)
	assert.Equal(t, 16.0, actual)
}

func Test_Metric_Trend(t *testing.T) {
	// Initialize a metric
	testMetric := NewMetric("test", nil)

	// Set now to a fixed time for testing
	nowFunc = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UTC
	defer func() {
		nowFunc = time.Now
	}()

	// Test with no records
	actual, err := testMetric.Trend(5 * time.Second)
	assert.Error(t, err)
	assert.Equal(t, 0.0, actual)

	// Insert test records
	for _, r := range getDefaultTestRecords() {
		testMetric.insertRecord(r.value, r.timestamp)
	}

	// Test trend with all records
	actual, err = testMetric.Trend(7 * time.Second)
	assert.NoError(t, err)
	assert.Equal(t, defaultRecordsSlope, actual)
}

func Test_Manager_Register_Unregister(t *testing.T) {
	mgr := NewManager()
	metricName := "sample"
	sample := NewMetric(metricName, nil)
	err := mgr.Register(sample)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(mgr.GetMetrics()))

	err = mgr.Register(sample)
	assert.Error(t, err)
	assert.Equal(t, 1, len(mgr.GetMetrics()))

	mgr.Unregister(metricName)
	assert.Equal(t, 0, len(mgr.GetMetrics()))

	// should not be a problem to unregister a non-existing metric
	mgr.Unregister(metricName)
	assert.Equal(t, 0, len(mgr.GetMetrics()))
}

func Test_Manager_GetMetric(t *testing.T) {
	mgr := NewManager()
	sample := NewMetric("sample", nil)
	err := mgr.Register(sample)
	assert.NoError(t, err)
	actual, err := mgr.GetMetric("sample")
	assert.NoError(t, err)
	assert.Equal(t, sample, actual)
	actual, err = mgr.GetMetric("invalid")
	assert.Error(t, err)
	assert.Nil(t, actual)
}

func Test_Manager_GetMetricsFromLabel(t *testing.T) {
	mgr := NewManager()

	sample := NewMetric("sample", nil)
	err := mgr.Register(sample)
	assert.NoError(t, err)

	foo := NewMetric("foo", []string{"foo"})
	err = mgr.Register(foo)
	assert.NoError(t, err)

	fooBar := NewMetric("foo_bar", []string{"foo", "bar"})
	err = mgr.Register(fooBar)
	assert.NoError(t, err)

	tests := map[string]struct {
		label    string
		expected map[string]*Metric
	}{
		"no label": {
			label: "",
			expected: map[string]*Metric{
				sample.name: sample,
				foo.name:    foo,
				fooBar.name: fooBar,
			},
		},
		"label not present": {
			label:    "not present",
			expected: map[string]*Metric{},
		},
		"2 matches": {
			label: "foo",
			expected: map[string]*Metric{
				foo.name:    foo,
				fooBar.name: fooBar,
			},
		},
		"1 match": {
			label: "bar",
			expected: map[string]*Metric{
				fooBar.name: fooBar,
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			actual := mgr.GetMetricsFromLabel(tc.label)
			assert.Equal(t, len(tc.expected), len(actual))
			assert.True(t, reflect.DeepEqual(actual, tc.expected))
		})
	}
}

func Test_Metric_getRecordsWithinTimeframe(t *testing.T) {
	m := NewMetric("testMetric", nil)
	nowFunc = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UTC
	defer func() {
		nowFunc = time.Now
	}()
	sampleRecords := getDefaultTestRecords()
	for _, r := range sampleRecords {
		m.insertRecord(r.value, r.timestamp)
	}
	tests := map[string]struct {
		duration time.Duration
		expected []record
	}{
		"all points": {
			duration: 5500 * time.Millisecond,
			expected: sampleRecords,
		},
		"all points with longer max duration": {
			duration: 10 * time.Second,
			expected: sampleRecords,
		},
		"short duration": {
			duration: 2500 * time.Millisecond,
			expected: []record{
				sampleRecords[3],
				sampleRecords[4],
				sampleRecords[5],
			},
		},
		"very short duration": {
			duration: 500 * time.Millisecond,
			expected: []record{
				sampleRecords[5],
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			actual := m.getRecordsWithinTimeframe(tc.duration)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func Test_convertRecordsToDataPoints(t *testing.T) {
	tests := map[string]struct {
		records  []record
		expected []math.DataPoint
	}{
		"all points": {
			records: getDefaultTestRecords(),
			expected: []math.DataPoint{
				{X: 0.0, Y: 10.0},
				{X: 1.0, Y: 12.0},
				{X: 2.0, Y: 11.0},
				{X: 3.0, Y: 13.0},
				{X: 4.0, Y: 15.0},
				{X: 5.0, Y: 16.0},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			actual := convertRecordsToDataPoints(tc.records)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func getDefaultTestRecords() []record {
	callTime := nowFunc()
	return []record{
		{10.0, callTime.Add(-5 * time.Second)},
		{12.0, callTime.Add(-4 * time.Second)},
		{11.0, callTime.Add(-3 * time.Second)},
		{13.0, callTime.Add(-2 * time.Second)},
		{15.0, callTime.Add(-1 * time.Second)},
		{16.0, callTime},
	}
}
