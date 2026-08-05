package lender

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	labelLendee = "lendee"
	labelType   = "type"
)

var lenderCreditsAllocated = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "lender_credits_allocated",
		Help: "The number of credits currently allocated to a lendee by the lender",
	},
	[]string{labelLendee, labelType},
)

var lenderCreditsMax = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "lender_credits_max",
		Help: "The maximum number of credits available for a lendee from the lender",
	},
	[]string{labelLendee, labelType},
)

func init() {
	if err := prometheus.Register(lenderCreditsAllocated); err != nil {
		logger.Debug().Err(err).Msgf("An error occurred registering lender_credits_allocated")
	}
	if err := prometheus.Register(lenderCreditsMax); err != nil {
		logger.Debug().Err(err).Msgf("An error occurred registering lender_credits_max")
	}
}

// publishLenderCreditsAllocatedMetric publishes the Prometheus metric for allocated credits for a lendee.
func publishLenderCreditsAllocatedMetric(lendeeName string, lendeeType string, credits float64) {
	labels := prometheus.Labels{
		labelLendee: lendeeName,
		labelType:   lendeeType,
	}
	if g, err := lenderCreditsAllocated.GetMetricWith(labels); err == nil {
		g.Set(credits)
	} else {
		logger.Debug().Err(err).Msgf("An error occurred publishing metric: %v", err)
	}
}

// publishLenderCreditsMaxMetric publishes the Prometheus metric for maximum credits for a lendee.
func publishLenderCreditsMaxMetric(lendeeName string, lendeeType string, maxCredits float64) {
	labels := prometheus.Labels{
		labelLendee: lendeeName,
		labelType:   lendeeType,
	}
	if g, err := lenderCreditsMax.GetMetricWith(labels); err == nil {
		g.Set(maxCredits)
	} else {
		logger.Debug().Err(err).Msgf("An error occurred publishing metric: %v", err)
	}
}

// removeLenderMetrics sets the prometheus metrics to zero for a lendee when they are unregistered.
func removeLenderMetrics(lendeeName string, lendeeType string) {
	publishLenderCreditsAllocatedMetric(lendeeName, lendeeType, 0.0)
	publishLenderCreditsMaxMetric(lendeeName, lendeeType, 0.0)
}
