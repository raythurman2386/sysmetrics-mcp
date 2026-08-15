package monitor

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Severity levels for alerts.
const (
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

// StatusHealthy is the aggregate health status reported when no thresholds
// have been crossed.
const StatusHealthy = "healthy"

// Alert is a single threshold crossing produced by the monitor.
type Alert struct {
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`
	Metric    string    `json:"metric"`
	Message   string    `json:"message"`
}

// ThresholdConfig configures the limits used to evaluate a snapshot.
type ThresholdConfig struct {
	// CPUHigh is the usage percent that triggers a warning.
	CPUHigh float64
	// CPUCritical is the usage percent that triggers a critical alert.
	CPUCritical float64
	// MemoryHigh and MemoryCritical mirror CPU thresholds for memory usage.
	MemoryHigh     float64
	MemoryCritical float64
	// DiskHigh and DiskCritical mirror CPU thresholds for disk usage.
	DiskHigh     float64
	DiskCritical float64
	// NetThroughputHighBytes is a per-interface warning threshold in bytes/sec.
	NetThroughputHighBytes float64
}

// DefaultThresholds returns sensible defaults for a typical host.
func DefaultThresholds() ThresholdConfig {
	return ThresholdConfig{
		CPUHigh:                80,
		CPUCritical:            95,
		MemoryHigh:             85,
		MemoryCritical:         95,
		DiskHigh:               85,
		DiskCritical:           95,
		NetThroughputHighBytes: 0,
	}
}

// EvaluateHealth inspects CPU/memory/disk usage and produces alerts plus an
// aggregate status. cpuPercent, memPercent, and diskPercent are usage
// percentages; netStats carries per-interface throughput in bytes/sec.
func EvaluateHealth(t ThresholdConfig, cpuPercent, memPercent, diskPercent float64, netStats []NetStat) (string, []Alert) {
	status := StatusHealthy
	var alerts []Alert

	status, alerts = applyPercentThreshold(status, alerts, threshold{t.CPUHigh, t.CPUCritical, "cpu", cpuPercent, "CPU usage"})
	status, alerts = applyPercentThreshold(status, alerts, threshold{t.MemoryHigh, t.MemoryCritical, "memory", memPercent, "Memory usage"})
	status, alerts = applyPercentThreshold(status, alerts, threshold{t.DiskHigh, t.DiskCritical, "disk", diskPercent, "Disk usage"})
	if t.NetThroughputHighBytes > 0 {
		for _, ns := range netStats {
			peak := ns.BytesSentPerSec
			if ns.BytesRecvPerSec > peak {
				peak = ns.BytesRecvPerSec
			}
			if peak > t.NetThroughputHighBytes {
				alerts = append(alerts, newAlert(SeverityWarning, "net", fmt.Sprintf(
					"High network throughput on %s (%.1f bytes/s)", ns.Interface, peak)))
			}
		}
	}

	return status, alerts
}

// threshold bundles the limits and metadata for a single metric.
type threshold struct {
	high     float64
	critical float64
	metric   string
	value    float64
	label    string
}

// applyPercentThreshold compares a percentage to the warning/critical limits
// and upgrades the overall status accordingly.
func applyPercentThreshold(status string, alerts []Alert, t threshold) (string, []Alert) {
	switch {
	case t.value > t.critical:
		status = maxSeverity(status, SeverityCritical)
		alerts = append(alerts, newAlert(SeverityCritical, "critical-"+t.metric, fmt.Sprintf("%s is critical (%.1f%%)", t.label, t.value)))
	case t.value > t.high:
		status = maxSeverity(status, SeverityWarning)
		alerts = append(alerts, newAlert(SeverityWarning, "high-"+t.metric, fmt.Sprintf("%s is high (%.1f%%)", t.label, t.value)))
	}
	return status, alerts
}

// maxSeverity returns the more severe of the two severity strings.
func maxSeverity(a, b string) string {
	if severityRank(a) >= severityRank(b) {
		return a
	}
	return b
}

func severityRank(s string) int {
	switch s {
	case SeverityCritical:
		return 2
	case SeverityWarning:
		return 1
	default:
		return 0
	}
}

func newAlert(severity, metric, message string) Alert {
	return Alert{
		Timestamp: time.Now(),
		Severity:  severity,
		Metric:    metric,
		Message:   message,
	}
}

// AlertBuffer is a concurrency-safe queue of generated alerts.
type AlertBuffer struct {
	mu      sync.Mutex
	alerts  []Alert
	maxSize int
}

// NewAlertBuffer creates an alert buffer retaining at most maxSize alerts.
func NewAlertBuffer(maxSize int) *AlertBuffer {
	if maxSize < 1 {
		maxSize = 1
	}
	return &AlertBuffer{alerts: make([]Alert, 0, maxSize), maxSize: maxSize}
}

// Add appends an alert, evicting the oldest if over capacity.
func (b *AlertBuffer) Add(a Alert) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.alerts = append(b.alerts, a)
	if len(b.alerts) > b.maxSize {
		b.alerts = b.alerts[1:]
	}
}

// All returns all buffered alerts in chronological order.
func (b *AlertBuffer) All() []Alert {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Alert, len(b.alerts))
	copy(out, b.alerts)
	return out
}

// ClearAndAll returns all alerts and empties the buffer (a read-then-drain).
func (b *AlertBuffer) ClearAndAll() []Alert {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Alert, len(b.alerts))
	copy(out, b.alerts)
	b.alerts = b.alerts[:0]
	return out
}

// SortAlerts orders alerts by timestamp ascending.
func SortAlerts(alerts []Alert) {
	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].Timestamp.Before(alerts[j].Timestamp)
	})
}
