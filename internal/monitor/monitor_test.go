package monitor

import (
	"context"
	"testing"
	"time"
)

func TestRate(t *testing.T) {
	if got := rate(100, 50, 2.0); got != 25.0 {
		t.Errorf("rate(100,50,2) = %v; want 25.0", got)
	}
	// Counter reset should yield zero, not a negative rate.
	if got := rate(10, 100, 2.0); got != 0 {
		t.Errorf("rate(10,100,2) after reset = %v; want 0", got)
	}
}

func TestSamplerBaselineAndDelta(t *testing.T) {
	s := NewSampler()

	// First call establishes a baseline with zero rates.
	snap1, err := s.Sample(context.Background())
	if err != nil {
		t.Fatalf("baseline sample error: %v", err)
	}
	if snap1.CPUPercent < 0 || snap1.CPUPercent > 100 {
		t.Errorf("unexpected baseline CPU percent: %v", snap1.CPUPercent)
	}

	// Second call should return delta rates without error.
	snap2, err := s.Sample(context.Background())
	if err != nil {
		t.Fatalf("delta sample error: %v", err)
	}
	if !snap2.Timestamp.After(snap1.Timestamp) {
		t.Errorf("delta timestamp not after baseline")
	}
	if snap2.MemPercent < 0 {
		t.Errorf("negative mem percent: %v", snap2.MemPercent)
	}
}

func TestSnapshotRingOrdering(t *testing.T) {
	ring := NewSnapshotRing(3)
	base := time.Now()

	for i := 0; i < 5; i++ {
		ring.Add(Snapshot{Timestamp: base.Add(time.Duration(i) * time.Second)})
	}

	all := ring.All()
	if len(all) != 3 {
		t.Fatalf("ring size = %d; want 3", len(all))
	}
	// Should retain the last 3, in chronological order (timestamps 2,3,4).
	for i, s := range all {
		want := base.Add(time.Duration(i+2) * time.Second)
		if !s.Timestamp.Equal(want) {
			t.Errorf("ring[%d] = %v; want %v", i, s.Timestamp, want)
		}
	}
}

func TestSnapshotRingSince(t *testing.T) {
	ring := NewSnapshotRing(10)
	base := time.Now()

	for i := 0; i < 4; i++ {
		ring.Add(Snapshot{Timestamp: base.Add(time.Duration(i) * time.Second)})
	}

	since := base.Add(2 * time.Second)
	got := ring.Since(since)
	if len(got) != 2 {
		t.Fatalf("Since() count = %d; want 2", len(got))
	}
}

func TestEvaluateHealthHealthy(t *testing.T) {
	tc := DefaultThresholds()
	status, alerts := EvaluateHealth(tc, 10, 30, 40, nil)
	if status != "healthy" {
		t.Errorf("status = %s; want healthy", status)
	}
	if len(alerts) != 0 {
		t.Errorf("unexpected alerts: %v", alerts)
	}
}

func TestEvaluateHealthWarningAndCritical(t *testing.T) {
	tc := DefaultThresholds()

	status, alerts := EvaluateHealth(tc, 90, 30, 40, nil)
	if status != SeverityWarning {
		t.Errorf("status = %s; want warning", status)
	}
	if len(alerts) == 0 {
		t.Error("expected a warning alert")
	}

	status, alerts = EvaluateHealth(tc, 99, 30, 40, nil)
	if status != SeverityCritical {
		t.Errorf("status = %s; want critical", status)
	}
	if len(alerts) == 0 {
		t.Error("expected a critical alert")
	}
}

func TestEvaluateHealthNetThroughput(t *testing.T) {
	tc := DefaultThresholds()
	tc.NetThroughputHighBytes = 1000

	// Below threshold: no alert.
	_, alerts := EvaluateHealth(tc, 10, 10, 10, []NetStat{{Interface: "eth0", BytesSentPerSec: 500}})
	if len(alerts) != 0 {
		t.Errorf("expected no alerts, got %v", alerts)
	}

	// Above threshold: warning alert.
	_, alerts = EvaluateHealth(tc, 10, 10, 10, []NetStat{{Interface: "eth0", BytesRecvPerSec: 5000}})
	if len(alerts) != 1 {
		t.Errorf("expected 1 alert, got %v", alerts)
	}
	if alerts[0].Severity != SeverityWarning {
		t.Errorf("severity = %s; want warning", alerts[0].Severity)
	}
}

func TestAlertBufferCapAndDrain(t *testing.T) {
	b := NewAlertBuffer(2)
	for i := 0; i < 3; i++ {
		b.Add(Alert{Metric: string(rune('a' + i))})
	}
	all := b.All()
	if len(all) != 2 {
		t.Fatalf("buffer size = %d; want 2", len(all))
	}
	// Oldest evicted; first retained should be 'b'.
	if all[0].Metric != "b" {
		t.Errorf("first retained = %q; want b", all[0].Metric)
	}

	drained := b.ClearAndAll()
	if len(drained) != 2 {
		t.Errorf("drained count = %d; want 2", len(drained))
	}
	if len(b.All()) != 0 {
		t.Error("buffer not empty after drain")
	}
}

func TestMonitorStartStop(t *testing.T) {
	m := NewMonitor(DefaultThresholds(), 10)

	if m.IsRunning() {
		t.Fatal("monitor should start stopped")
	}
	if !m.Start(time.Second) {
		t.Fatal("first Start should return true")
	}
	// Second start should be a no-op.
	if m.Start(time.Second) {
		t.Fatal("second Start should return false")
	}
	if !m.IsRunning() {
		t.Fatal("monitor should be running")
	}

	// SampleNow works even while running.
	snap, err := m.SampleNow(context.Background())
	if err != nil {
		t.Fatalf("SampleNow error: %v", err)
	}
	if snap.CPUPercent < 0 {
		t.Errorf("negative CPU percent: %v", snap.CPUPercent)
	}

	m.Stop()
	if m.IsRunning() {
		t.Fatal("monitor should be stopped")
	}
	// Stopping again should be a no-op (no panic).
	m.Stop()
}
