package metrics

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
)

var (
	// ManagerKey is the plugin key for the metrics manager.
	ManagerKey = key{Name: "gopher-hole/metrics/manager"}
)

type Manager struct {
	metrics map[string]*Metric
	mutex   sync.Mutex
}

// NewManager initializes a new Manager struct.
func NewManager() *Manager {
	return &Manager{
		metrics: make(map[string]*Metric),
		mutex:   sync.Mutex{},
	}
}

// Register adds a new Metric to the Manager.
func (mgr *Manager) Register(metric *Metric) error {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()
	if _, ok := mgr.metrics[metric.name]; ok {
		return fmt.Errorf("metric %s already exists", metric.name)
	}
	mgr.metrics[metric.name] = metric
	return nil
}

// Unregister removes a Metric from the Manager.
func (mgr *Manager) Unregister(metricName string) {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()
	delete(mgr.metrics, metricName)
}

// GetMetrics returns a map off all Metric structs registered to the Manager.
func (mgr *Manager) GetMetrics() map[string]*Metric {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()
	// Make a copy of the metrics map to return so we can't modify the Manager's actual map
	var metricsCopy = make(map[string]*Metric)
	maps.Copy(metricsCopy, mgr.metrics)
	return metricsCopy
}

// GetMetricsFromLabel returns a map off all Metric structs registered to the Manager
// that contain the provided label.
func (mgr *Manager) GetMetricsFromLabel(label string) map[string]*Metric {
	metrics := mgr.GetMetrics()
	if label == "" {
		return metrics
	}
	labeledMetrics := make(map[string]*Metric)
	for name, metric := range metrics {
		if slices.Contains(metric.labels, label) {
			labeledMetrics[name] = metric
		}
	}
	return labeledMetrics
}

// GetMetric returns the Metric struct that is stored with the provided name.
func (mgr *Manager) GetMetric(name string) (*Metric, error) {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()
	metric, ok := mgr.metrics[name]
	if !ok {
		return nil, fmt.Errorf("metric %s not found", name)
	}
	return metric, nil
}

// NewManagerContext adds the metrics.Manager to the existing context
func NewManagerContext(ctx context.Context, mgr *Manager) context.Context {
	return context.WithValue(ctx, ManagerKey, mgr)
}

// GetManagerFromContext retrieves the metrics.Manager from the context
func GetManagerFromContext(ctx context.Context) (*Manager, error) {
	mgr, ok := ctx.Value(ManagerKey).(*Manager)
	if !ok {
		return nil, errors.New("metrics manager not found in context")
	}
	return mgr, nil
}
