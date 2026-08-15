// Package monitor implements delta-based metric sampling and threshold
// evaluation for the SysMetrics MCP server. It computes per-second rates for
// CPU, network, and disk I/O by diffing cumulative counters between samples.
package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// NetStat is the per-second network throughput for a single interface.
type NetStat struct {
	Interface       string  `json:"interface"`
	BytesSentPerSec float64 `json:"bytes_sent_per_sec"`
	BytesRecvPerSec float64 `json:"bytes_recv_per_sec"`
}

// DiskIOStat is the per-second disk I/O rate for a single device.
type DiskIOStat struct {
	Device           string  `json:"device"`
	ReadBytesPerSec  float64 `json:"read_bytes_per_sec"`
	WriteBytesPerSec float64 `json:"write_bytes_per_sec"`
	ReadIOPS         float64 `json:"read_iops"`
	WriteIOPS        float64 `json:"write_iops"`
}

// Snapshot holds the metrics collected on a single sampling tick.
type Snapshot struct {
	Timestamp   time.Time    `json:"timestamp"`
	CPUPercent  float64      `json:"cpu_percent"`
	MemPercent  float64      `json:"mem_percent"`
	DiskPercent float64      `json:"disk_percent"`
	Net         []NetStat    `json:"net"`
	DiskIO      []DiskIOStat `json:"disk_io"`
}

// Sampler computes delta-based rates from cumulative gopsutil counters.
// It is safe for concurrent use by a monitoring goroutine and tool handlers.
type Sampler struct {
	mu sync.Mutex

	lastNet  map[string]net.IOCountersStat
	lastDisk map[string]disk.IOCountersStat
	lastTime time.Time
	hasLast  bool
}

// NewSampler creates a Sampler with no prior state.
func NewSampler() *Sampler {
	return &Sampler{}
}

// Sample collects a new snapshot, computing per-second rates by diffing the
// previous sample's cumulative counters against the current ones. The first
// call establishes a baseline and reports zero rates.
func (s *Sampler) Sample(ctx context.Context) (Snapshot, error) {
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil || len(cpuPercent) == 0 {
		cpuPercent = []float64{0}
	}

	netIO, err := net.IOCounters(true)
	if err != nil {
		netIO = []net.IOCountersStat{}
	}

	diskIO, err := disk.IOCounters()
	if err != nil {
		diskIO = map[string]disk.IOCountersStat{}
	}

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		memInfo = &mem.VirtualMemoryStat{}
	}

	diskUsage, err := disk.Usage("/")
	if err != nil {
		diskUsage = &disk.UsageStat{}
	}

	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	elapsed := now.Sub(s.lastTime).Seconds()
	if !s.hasLast || elapsed <= 0 {
		// Baseline sample: record counters and return zero rates.
		s.lastNet = snapshotNet(netIO)
		s.lastDisk = diskIO
		s.lastTime = now
		s.hasLast = true

		return Snapshot{
			Timestamp:   now,
			CPUPercent:  cpuPercent[0],
			MemPercent:  memInfo.UsedPercent,
			DiskPercent: diskUsage.UsedPercent,
			Net:         buildNetStats(netIO, s.lastNet, 0),
			DiskIO:      buildDiskStats(diskIO, s.lastDisk, 0),
		}, nil
	}

	prevNet := s.lastNet
	prevDisk := s.lastDisk
	s.lastNet = snapshotNet(netIO)
	s.lastDisk = diskIO
	s.lastTime = now

	return Snapshot{
		Timestamp:   now,
		CPUPercent:  cpuPercent[0],
		MemPercent:  memInfo.UsedPercent,
		DiskPercent: diskUsage.UsedPercent,
		Net:         buildNetStats(netIO, prevNet, elapsed),
		DiskIO:      buildDiskStats(diskIO, prevDisk, elapsed),
	}, nil
}

// snapshotNet converts a slice of net counters into a lookup map.
func snapshotNet(ioStats []net.IOCountersStat) map[string]net.IOCountersStat {
	result := make(map[string]net.IOCountersStat, len(ioStats))
	for _, io := range ioStats {
		result[io.Name] = io
	}
	return result
}

func buildNetStats(current []net.IOCountersStat, prev map[string]net.IOCountersStat, elapsed float64) []NetStat {
	result := make([]NetStat, 0, len(current))
	for _, io := range current {
		stat := NetStat{Interface: io.Name}
		if prevIO, ok := prev[io.Name]; ok && elapsed > 0 {
			stat.BytesSentPerSec = rate(io.BytesSent, prevIO.BytesSent, elapsed)
			stat.BytesRecvPerSec = rate(io.BytesRecv, prevIO.BytesRecv, elapsed)
		}
		result = append(result, stat)
	}
	return result
}

func buildDiskStats(current map[string]disk.IOCountersStat, prev map[string]disk.IOCountersStat, elapsed float64) []DiskIOStat {
	result := make([]DiskIOStat, 0, len(current))
	for name, io := range current {
		stat := DiskIOStat{Device: name}
		if prevIO, ok := prev[name]; ok && elapsed > 0 {
			stat.ReadBytesPerSec = rate(io.ReadBytes, prevIO.ReadBytes, elapsed)
			stat.WriteBytesPerSec = rate(io.WriteBytes, prevIO.WriteBytes, elapsed)
			stat.ReadIOPS = rate(io.ReadCount, prevIO.ReadCount, elapsed)
			stat.WriteIOPS = rate(io.WriteCount, prevIO.WriteCount, elapsed)
		}
		result = append(result, stat)
	}
	return result
}

// rate returns the per-second rate between two cumulative counters.
func rate(current, previous uint64, elapsed float64) float64 {
	if current < previous {
		return 0
	}
	return float64(current-previous) / elapsed
}
