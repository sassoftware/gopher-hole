package lender

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getGaugeValue(gaugeVec *prometheus.GaugeVec, labels prometheus.Labels) (float64, error) {
	gauge, err := gaugeVec.GetMetricWith(labels)
	if err != nil {
		return 0, err
	}
	m := &dto.Metric{}
	if err := gauge.Write(m); err != nil {
		return 0, err
	}
	return m.GetGauge().GetValue(), nil
}

func TestPublishLenderCreditsAllocatedMetric(t *testing.T) {
	lendeeName := "test_consumer_allocated"
	lendeeType := "testGroup"

	publishLenderCreditsAllocatedMetric(lendeeName, lendeeType, 3.0)

	value, err := getGaugeValue(lenderCreditsAllocated, prometheus.Labels{labelLendee: lendeeName, labelType: lendeeType})
	require.NoError(t, err)
	assert.Equal(t, 3.0, value)

	// Update the value
	publishLenderCreditsAllocatedMetric(lendeeName, lendeeType, 5.0)

	value, err = getGaugeValue(lenderCreditsAllocated, prometheus.Labels{labelLendee: lendeeName, labelType: lendeeType})
	require.NoError(t, err)
	assert.Equal(t, 5.0, value)
}

func TestPublishLenderCreditsMaxMetric(t *testing.T) {
	lendeeName := "test_consumer_max"
	lendeeType := "testGroup"

	publishLenderCreditsMaxMetric(lendeeName, lendeeType, 10.0)

	value, err := getGaugeValue(lenderCreditsMax, prometheus.Labels{labelLendee: lendeeName, labelType: lendeeType})
	require.NoError(t, err)
	assert.Equal(t, 10.0, value)
}

func TestLenderPrometheusMetrics_MultipleConsumers(t *testing.T) {
	consumers := []struct {
		name       string
		lendeeType string
		allocated  float64
		max        float64
	}{
		{"consumer_a", "typeA", 2.0, 8.0},
		{"consumer_b", "typeB", 5.0, 10.0},
		{"consumer_c", "typeC", 0.0, 3.0},
	}

	for _, c := range consumers {
		publishLenderCreditsAllocatedMetric(c.name, c.lendeeType, c.allocated)
		publishLenderCreditsMaxMetric(c.name, c.lendeeType, c.max)
	}

	for _, c := range consumers {
		allocValue, err := getGaugeValue(lenderCreditsAllocated, prometheus.Labels{labelLendee: c.name, labelType: c.lendeeType})
		require.NoError(t, err)
		assert.Equal(t, c.allocated, allocValue, "allocated credits mismatch for %s", c.name)

		maxValue, err := getGaugeValue(lenderCreditsMax, prometheus.Labels{labelLendee: c.name, labelType: c.lendeeType})
		require.NoError(t, err)
		assert.Equal(t, c.max, maxValue, "max credits mismatch for %s", c.name)
	}
}
