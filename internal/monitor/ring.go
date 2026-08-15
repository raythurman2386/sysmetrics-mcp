package monitor

import (
	"sort"
	"sync"
	"time"
)

// SnapshotRing is a fixed-capacity, concurrency-safe circular buffer of
// metric snapshots. It keeps the most recent N samples for history queries.
type SnapshotRing struct {
	mu    sync.RWMutex
	ring  []Snapshot
	size  int
	head  int
	count int
}

// NewSnapshotRing creates a ring buffer holding at most size snapshots.
// size is clamped to a minimum of 1.
func NewSnapshotRing(size int) *SnapshotRing {
	if size < 1 {
		size = 1
	}
	return &SnapshotRing{
		ring: make([]Snapshot, size),
		size: size,
	}
}

// Add appends a snapshot, evicting the oldest if the buffer is full.
func (r *SnapshotRing) Add(s Snapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ring[r.head] = s
	r.head = (r.head + 1) % r.size
	if r.count < r.size {
		r.count++
	}
}

// All returns the buffered snapshots in chronological order (oldest first).
func (r *SnapshotRing) All() []Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Snapshot, 0, r.count)
	// Oldest element is r.head - r.count (wrapping).
	start := (r.head - r.count + r.size) % r.size
	for i := 0; i < r.count; i++ {
		idx := (start + i) % r.size
		out = append(out, r.ring[idx])
	}
	return out
}

// Since returns snapshots with a timestamp at or after the given time.
func (r *SnapshotRing) Since(since time.Time) []Snapshot {
	all := r.All()
	out := make([]Snapshot, 0, len(all))
	for _, s := range all {
		if !s.Timestamp.Before(since) {
			out = append(out, s)
		}
	}
	return out
}

// SortSnapshots orders snapshots by timestamp ascending. Provided for callers
// that assemble snapshots outside the ring.
func SortSnapshots(snaps []Snapshot) {
	sort.Slice(snaps, func(i, j int) bool {
		return snaps[i].Timestamp.Before(snaps[j].Timestamp)
	})
}
