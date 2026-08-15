package monitor

import (
	"context"
	"sync"
	"time"
)

// Monitor is the coordinator that periodically samples system metrics,
// retains recent history, and evaluates thresholds to generate alerts.
type Monitor struct {
	sampler    *Sampler
	history    *SnapshotRing
	alerts     *AlertBuffer
	thresholds ThresholdConfig

	mu         sync.Mutex
	interval   time.Duration
	cancel     context.CancelFunc
	running    bool
	lastStatus string
}

// NewMonitor creates a Monitor with the given thresholds and history capacity.
func NewMonitor(thresholds ThresholdConfig, historySize int) *Monitor {
	return &Monitor{
		sampler:    NewSampler(),
		history:    NewSnapshotRing(historySize),
		alerts:     NewAlertBuffer(100),
		thresholds: thresholds,
	}
}

// Start begins periodic sampling. It is a no-op if already running.
func (m *Monitor) Start(interval time.Duration) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		return false
	}
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.interval = interval
	m.cancel = cancel
	m.running = true
	go m.run(ctx, interval)
	return true
}

// Stop halts periodic sampling and waits for the goroutine to finish.
func (m *Monitor) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	cancel := m.cancel
	m.running = false
	m.cancel = nil
	m.mu.Unlock()

	cancel()
}

// IsRunning reports whether the monitor is currently sampling.
func (m *Monitor) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// Interval returns the configured sampling interval.
func (m *Monitor) Interval() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.interval
}

// run ticks on the interval until the context is cancelled.
func (m *Monitor) run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.tick(ctx)
		}
	}
}

// tick samples once and evaluates thresholds.
func (m *Monitor) tick(ctx context.Context) {
	snap, err := m.sampler.Sample(ctx)
	if err != nil {
		return
	}
	m.history.Add(snap)
	m.evaluate(snap)
}

// evaluate runs threshold evaluation and stores any resulting alerts.
func (m *Monitor) evaluate(snap Snapshot) {
	status, alerts := EvaluateHealth(m.thresholds, snap.CPUPercent, snap.MemPercent, snap.DiskPercent, snap.Net)
	m.mu.Lock()
	m.lastStatus = status
	m.mu.Unlock()
	for _, a := range alerts {
		m.alerts.Add(a)
	}
}

// Status returns the most recently evaluated aggregate health status.
func (m *Monitor) Status() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lastStatus == "" {
		return StatusHealthy
	}
	return m.lastStatus
}

// History returns all buffered snapshots in chronological order.
func (m *Monitor) History() []Snapshot {
	return m.history.All()
}

// Alerts returns all buffered alerts.
func (m *Monitor) Alerts() []Alert {
	return m.alerts.All()
}

// DrainAlerts returns and clears all buffered alerts.
func (m *Monitor) DrainAlerts() []Alert {
	return m.alerts.ClearAndAll()
}

// Last returns the most recent snapshot, or a zero snapshot if none exists.
func (m *Monitor) Last() (Snapshot, bool) {
	h := m.history.All()
	if len(h) == 0 {
		return Snapshot{}, false
	}
	return h[len(h)-1], true
}

// SampleNow collects an immediate snapshot (useful for one-shot queries).
func (m *Monitor) SampleNow(ctx context.Context) (Snapshot, error) {
	return m.sampler.Sample(ctx)
}
