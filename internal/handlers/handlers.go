// Package handlers implements the MCP tool handlers for system metrics.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"time"

	"sysmetrics-mcp/internal/config"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// HandlerManager manages the MCP tool handlers
type HandlerManager struct {
	cfg *config.Config
}

// NewHandlerManager creates a new HandlerManager
func NewHandlerManager(cfg *config.Config) *HandlerManager {
	return &HandlerManager{cfg: cfg}
}

// RegisterTools registers all available tools with the MCP server
func (h *HandlerManager) RegisterTools(s *server.MCPServer) {
	// System info tool
	s.AddTool(mcp.NewTool("get_system_info",
		mcp.WithDescription("Get system information including hostname, OS, uptime, and platform details")),
		h.HandleGetSystemInfo)

	// CPU metrics tool
	s.AddTool(mcp.NewTool("get_cpu_metrics",
		mcp.WithDescription("Get CPU usage, temperature, and load average"),
		mcp.WithString("temp_unit", mcp.Description("Override temperature unit: celsius, fahrenheit, or kelvin"),
			mcp.Enum(config.UnitCelsius, config.UnitFahrenheit, config.UnitKelvin))),
		h.HandleGetCPUMetrics)

	// Memory metrics tool
	s.AddTool(mcp.NewTool("get_memory_metrics",
		mcp.WithDescription("Get memory usage statistics including RAM and swap")),
		h.HandleGetMemoryMetrics)

	// Disk metrics tool
	s.AddTool(mcp.NewTool("get_disk_metrics",
		mcp.WithDescription("Get disk usage statistics for mount points"),
		mcp.WithString("mount_points", mcp.Description("Comma-separated mount points to check (overrides config default)")),
		mcp.WithBoolean("human_readable", mcp.Description("Include human-readable sizes alongside bytes"))),
		h.HandleGetDiskMetrics)

	// Network metrics tool
	s.AddTool(mcp.NewTool("get_network_metrics",
		mcp.WithDescription("Get network interface statistics"),
		mcp.WithString("interfaces", mcp.Description("Comma-separated interface names to check (overrides config default)"))),
		h.HandleGetNetworkMetrics)

	// Process list tool
	s.AddTool(mcp.NewTool("get_process_list",
		mcp.WithDescription("Get list of running processes sorted by resource usage"),
		mcp.WithNumber("limit", mcp.Description("Maximum number of processes to return (overrides config default)")),
		mcp.WithString("sort_by", mcp.Description("Sort by: cpu, memory, or pid"),
			mcp.Enum("cpu", "memory", "pid"))),
		h.HandleGetProcessList)

	// Thermal status tool
	s.AddTool(mcp.NewTool("get_thermal_status",
		mcp.WithDescription("Get thermal status including temperatures and throttling information"),
		mcp.WithString("temp_unit", mcp.Description("Override temperature unit: celsius, fahrenheit, or kelvin"),
			mcp.Enum(config.UnitCelsius, config.UnitFahrenheit, config.UnitKelvin))),
		h.HandleGetThermalStatus)
}

// HandleGetSystemInfo returns system information
func (h *HandlerManager) HandleGetSystemInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	info, err := host.Info()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get system info: %v", err)), nil
	}

	// Uptime is uint64, but Duration takes int64.
	// This will only overflow if uptime > 292 years, which is acceptable.
	//nolint:gosec // G115: integer overflow conversion safe for reasonable uptimes
	uptime := time.Duration(info.Uptime) * time.Second

	result := map[string]interface{}{
		"hostname":         info.Hostname,
		"os":               info.OS,
		"platform":         info.Platform,
		"platform_family":  info.PlatformFamily,
		"platform_version": info.PlatformVersion,
		"kernel_version":   info.KernelVersion,
		"kernel_arch":      info.KernelArch,
		"uptime_seconds":   info.Uptime,
		"uptime_human":     uptime.String(),
		// BootTime is unix timestamp (uint64). Standard unix time fits in int64 until year 2038+ (actually much later for 64-bit).
		//nolint:gosec // G115: integer overflow conversion safe for standard unix timestamps
		"boot_time":  time.Unix(int64(info.BootTime), 0).Format(time.RFC3339),
		"procs":      info.Procs,
		"go_version": runtime.Version(),
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// HandleGetCPUMetrics returns CPU metrics
func (h *HandlerManager) HandleGetCPUMetrics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get temperature unit from args or config
	tempUnit := h.cfg.TempUnit
	if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
		if unit, ok := args["temp_unit"].(string); ok && unit != "" {
			tempUnit = strings.ToLower(unit)
		}
	}

	// Get CPU usage
	percentages, err := cpu.Percent(0, false)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get CPU usage: %v", err)), nil
	}

	// Get per-CPU usage
	perCPU, err := cpu.Percent(0, true)
	if err != nil {
		perCPU = []float64{}
	}

	// Get CPU info
	cpuInfo, err := cpu.Info()
	if err != nil {
		cpuInfo = []cpu.InfoStat{}
	}

	// Get load average
	loadAvg, err := load.Avg()
	if err != nil {
		loadAvg = &load.AvgStat{}
	}

	// Get CPU temperature
	tempCelsius, hasTemp := config.GetRaspberryPiTemp()
	temps := config.ConvertTemperature(tempCelsius, tempUnit)

	result := map[string]interface{}{
		"usage_percent":         percentages[0],
		"per_cpu_percent":       perCPU,
		"core_count":            len(perCPU),
		"physical_cores":        runtime.NumCPU(),
		"load_average":          map[string]float64{"1min": loadAvg.Load1, "5min": loadAvg.Load5, "15min": loadAvg.Load15},
		"temperature_celsius":   tempCelsius,
		"temperature_converted": temps,
		"temperature_unit":      tempUnit,
		"has_temperature":       hasTemp,
	}

	if len(cpuInfo) > 0 {
		result["model"] = cpuInfo[0].ModelName
		result["mhz"] = cpuInfo[0].Mhz
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// HandleGetMemoryMetrics returns memory metrics
func (h *HandlerManager) HandleGetMemoryMetrics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get memory info: %v", err)), nil
	}

	swapInfo, err := mem.SwapMemory()
	if err != nil {
		swapInfo = &mem.SwapMemoryStat{}
	}

	result := map[string]interface{}{
		"ram": map[string]interface{}{
			"total_bytes":     memInfo.Total,
			"total_human":     config.BytesToHuman(memInfo.Total),
			"available_bytes": memInfo.Available,
			"available_human": config.BytesToHuman(memInfo.Available),
			"used_bytes":      memInfo.Used,
			"used_human":      config.BytesToHuman(memInfo.Used),
			"free_bytes":      memInfo.Free,
			"free_human":      config.BytesToHuman(memInfo.Free),
			"usage_percent":   memInfo.UsedPercent,
			"buffers_bytes":   memInfo.Buffers,
			"cached_bytes":    memInfo.Cached,
		},
		"swap": map[string]interface{}{
			"total_bytes":   swapInfo.Total,
			"total_human":   config.BytesToHuman(swapInfo.Total),
			"used_bytes":    swapInfo.Used,
			"used_human":    config.BytesToHuman(swapInfo.Used),
			"free_bytes":    swapInfo.Free,
			"free_human":    config.BytesToHuman(swapInfo.Free),
			"usage_percent": swapInfo.UsedPercent,
		},
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// HandleGetDiskMetrics returns disk metrics
func (h *HandlerManager) HandleGetDiskMetrics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get mount points from args or config
	mountPoints := h.cfg.MountPoints
	humanReadable := true

	if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
		if mpStr, ok := args["mount_points"].(string); ok && mpStr != "" {
			mountPoints = config.SplitAndTrim(mpStr)
		}
		if hr, ok := args["human_readable"].(bool); ok {
			humanReadable = hr
		}
	}

	// If no mount points specified, get all partitions
	if len(mountPoints) == 0 {
		partitions, err := disk.Partitions(false)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get disk partitions: %v", err)), nil
		}
		for _, p := range partitions {
			// Skip special filesystems
			if p.Fstype == "tmpfs" || p.Fstype == "devtmpfs" || p.Fstype == "squashfs" {
				continue
			}
			mountPoints = append(mountPoints, p.Mountpoint)
		}
	}

	diskData := []map[string]interface{}{}
	for _, mp := range mountPoints {
		usage, err := disk.Usage(mp)
		if err != nil {
			continue
		}

		diskInfo := map[string]interface{}{
			"mount_point":   mp,
			"total_bytes":   usage.Total,
			"used_bytes":    usage.Used,
			"free_bytes":    usage.Free,
			"usage_percent": usage.UsedPercent,
			"fstype":        usage.Fstype,
		}

		if humanReadable {
			diskInfo["total_human"] = config.BytesToHuman(usage.Total)
			diskInfo["used_human"] = config.BytesToHuman(usage.Used)
			diskInfo["free_human"] = config.BytesToHuman(usage.Free)
		}

		diskData = append(diskData, diskInfo)
	}

	result := map[string]interface{}{
		"disks": diskData,
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// HandleGetNetworkMetrics returns network metrics
func (h *HandlerManager) HandleGetNetworkMetrics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get interfaces from args or config
	interfaces := h.cfg.Interfaces

	if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
		if ifStr, ok := args["interfaces"].(string); ok && ifStr != "" {
			interfaces = config.SplitAndTrim(ifStr)
		}
	}

	// Get all network stats
	netIO, err := net.IOCounters(true)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get network stats: %v", err)), nil
	}

	// Get interface addresses
	interfacesList, err := net.Interfaces()
	if err != nil {
		interfacesList = []net.InterfaceStat{}
	}

	// Build interface address map
	addrMap := make(map[string][]string)
	for _, iface := range interfacesList {
		var addrs []string
		for _, addr := range iface.Addrs {
			addrs = append(addrs, addr.Addr)
		}
		addrMap[iface.Name] = addrs
	}

	// Filter and format results
	netData := []map[string]interface{}{}
	for _, io := range netIO {
		// Skip loopback by default unless explicitly requested
		if io.Name == "lo" && !contains(interfaces, "lo") {
			continue
		}

		// If specific interfaces requested, filter
		if len(interfaces) > 0 && !contains(interfaces, io.Name) {
			continue
		}

		netInfo := map[string]interface{}{
			"interface":    io.Name,
			"bytes_sent":   io.BytesSent,
			"bytes_recv":   io.BytesRecv,
			"packets_sent": io.PacketsSent,
			"packets_recv": io.PacketsRecv,
			"errors_in":    io.Errin,
			"errors_out":   io.Errout,
			"drops_in":     io.Dropin,
			"drops_out":    io.Dropout,
			"ip_addresses": addrMap[io.Name],
		}

		netData = append(netData, netInfo)
	}

	result := map[string]interface{}{
		"interfaces": netData,
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// HandleGetProcessList returns process list
func (h *HandlerManager) HandleGetProcessList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	limit := h.cfg.MaxProcesses
	sortBy := "cpu"

	if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
		if l, ok := args["limit"].(float64); ok && l > 0 {
			limit = int(l)
			if limit > 50 {
				limit = 50
			}
		}
		if s, ok := args["sort_by"].(string); ok && s != "" {
			sortBy = strings.ToLower(s)
		}
	}

	processes, err := process.Processes()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get processes: %v", err)), nil
	}

	type procInfo struct {
		PID        int32    `json:"pid"`
		Name       string   `json:"name"`
		CPU        float64  `json:"cpu_percent"`
		Memory     float32  `json:"memory_percent"`
		RSS        uint64   `json:"rss_bytes"`
		Status     []string `json:"status"`
		CreateTime int64    `json:"create_time"`
	}

	procList := []procInfo{}
	for _, p := range processes {
		name, _ := p.Name()
		cpu, _ := p.CPUPercent()
		mem, _ := p.MemoryPercent()
		memInfo, _ := p.MemoryInfo()
		status, _ := p.Status()
		createTime, _ := p.CreateTime()

		procList = append(procList, procInfo{
			PID:        p.Pid,
			Name:       name,
			CPU:        cpu,
			Memory:     mem,
			RSS:        memInfo.RSS,
			Status:     status,
			CreateTime: createTime / 1000, // Convert from ms to seconds
		})
	}

	// Sort based on criteria
	switch sortBy {
	case "memory":
		sort.Slice(procList, func(i, j int) bool {
			return procList[i].Memory > procList[j].Memory
		})
	case "pid":
		sort.Slice(procList, func(i, j int) bool {
			return procList[i].PID < procList[j].PID
		})
	default: // cpu
		sort.Slice(procList, func(i, j int) bool {
			return procList[i].CPU > procList[j].CPU
		})
	}

	// Limit results
	if len(procList) > limit {
		procList = procList[:limit]
	}

	result := map[string]interface{}{
		"processes": procList,
		"total":     len(processes),
		"shown":     len(procList),
		"sort_by":   sortBy,
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// HandleGetThermalStatus returns thermal status
func (h *HandlerManager) HandleGetThermalStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tempUnit := h.cfg.TempUnit

	if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
		if unit, ok := args["temp_unit"].(string); ok && unit != "" {
			tempUnit = strings.ToLower(unit)
		}
	}

	// Get CPU temperature
	cpuTempC, hasCPUTemp := config.GetRaspberryPiTemp()

	// Get GPU temperature (Pi-specific)
	var gpuTempC float64
	var hasGPUTemp bool
	if h.cfg.EnableGPU {
		gpuTempC, hasGPUTemp = config.GetRaspberryPiGPUTemp()
	}

	// Get throttling status (Pi-specific)
	var throttleStatus map[string]interface{}
	hasThrottleStatus := false
	if h.cfg.EnableGPU {
		throttleStatus, hasThrottleStatus = config.GetThrottledStatus()
	}

	result := map[string]interface{}{
		"cpu_temperature": map[string]interface{}{
			"available": hasCPUTemp,
			"celsius":   cpuTempC,
			"converted": config.ConvertTemperature(cpuTempC, tempUnit),
			"unit":      tempUnit,
		},
		"gpu_temperature": map[string]interface{}{
			"available": hasGPUTemp,
		},
		"throttling": map[string]interface{}{
			"available": hasThrottleStatus,
		},
		"platform": "raspberry_pi",
	}

	if hasGPUTemp {
		result["gpu_temperature"].(map[string]interface{})["celsius"] = gpuTempC
		result["gpu_temperature"].(map[string]interface{})["converted"] = config.ConvertTemperature(gpuTempC, tempUnit)
	}

	if hasThrottleStatus {
		result["throttling"].(map[string]interface{})["status"] = throttleStatus
	} else {
		result["platform"] = "generic_linux"
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// contains checks if a string slice contains a value
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
