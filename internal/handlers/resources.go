package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"sysmetrics-mcp/internal/config"
	"sysmetrics-mcp/internal/monitor"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

const (
	resourceOverview  = "sys://metrics/overview"
	resourceCPU       = "sys://metrics/cpu"
	resourceMemory    = "sys://metrics/memory"
	resourceDisk      = "sys://metrics/disk"
	resourceNetwork   = "sys://metrics/network"
	resourceProcesses = "sys://metrics/processes/top"
)

// RegisterResources registers MCP resources and resource templates.
func (h *HandlerManager) RegisterResources(s *server.MCPServer) {
	s.AddResource(mcp.NewResource(resourceOverview, "System Overview",
		mcp.WithResourceDescription("Aggregated system health dashboard"),
		mcp.WithMIMEType("application/json")),
		h.readOverview)

	s.AddResource(mcp.NewResource(resourceCPU, "CPU Metrics",
		mcp.WithResourceDescription("Current CPU usage and load average"),
		mcp.WithMIMEType("application/json")),
		h.readCPU)

	s.AddResource(mcp.NewResource(resourceMemory, "Memory Metrics",
		mcp.WithResourceDescription("Current RAM and swap usage"),
		mcp.WithMIMEType("application/json")),
		h.readMemory)

	s.AddResource(mcp.NewResource(resourceDisk, "Disk Metrics",
		mcp.WithResourceDescription("Disk usage for all mount points"),
		mcp.WithMIMEType("application/json")),
		h.readDisk)

	s.AddResource(mcp.NewResource(resourceNetwork, "Network Metrics",
		mcp.WithResourceDescription("Current network interface statistics"),
		mcp.WithMIMEType("application/json")),
		h.readNetwork)

	s.AddResourceTemplate(mcp.NewResourceTemplate(resourceProcesses, "Top Processes",
		mcp.WithTemplateDescription("Top N processes by resource usage (limit parameter)"),
		mcp.WithTemplateMIMEType("application/json")),
		h.readProcesses)
}

// resourceText wraps a marshaled payload into an MCP text resource contents.
func resourceText(uri string, payload interface{}) ([]mcp.ResourceContents, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return []mcp.ResourceContents{mcp.TextResourceContents{
		URI:      uri,
		MIMEType: "application/json",
		Text:     string(bytes),
	}}, nil
}

func (h *HandlerManager) readOverview(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		cpuPercent = []float64{0}
	}

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	rootDisk, err := disk.Usage("/")
	if err != nil {
		return nil, fmt.Errorf("failed to get root disk info: %w", err)
	}

	info, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}

	status, alerts := monitor.EvaluateHealth(
		monitor.DefaultThresholds(),
		cpuPercent[0],
		memInfo.UsedPercent,
		rootDisk.UsedPercent,
		[]monitor.NetStat{},
	)

	payload := map[string]interface{}{
		"status":   status,
		"warnings": alerts,
		"cpu": map[string]interface{}{
			"usage_percent": cpuPercent[0],
		},
		"memory": map[string]interface{}{
			"usage_percent":   memInfo.UsedPercent,
			"available_human": config.BytesToHuman(memInfo.Available),
			"total_human":     config.BytesToHuman(memInfo.Total),
		},
		"disk": map[string]interface{}{
			"mount_point":   "/",
			"usage_percent": rootDisk.UsedPercent,
			"free_human":    config.BytesToHuman(rootDisk.Free),
		},
		"hostname": info.Hostname,
	}

	return resourceText(request.Params.URI, payload)
}

func (h *HandlerManager) readCPU(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	cpuInfo, err := cpu.Info()
	if err != nil {
		cpuInfo = []cpu.InfoStat{}
	}

	model := ""
	mhz := float64(0)
	cores := int32(0)
	if len(cpuInfo) > 0 {
		model = cpuInfo[0].ModelName
		mhz = cpuInfo[0].Mhz
		cores = cpuInfo[0].Cores
	}

	temp, hasTemp := config.GetRaspberryPiTemp()

	payload := map[string]interface{}{
		"model":               model,
		"mhz":                 mhz,
		"core_count":          cores,
		"temperature_celsius": temp,
		"has_temperature":     hasTemp,
	}

	return resourceText(request.Params.URI, payload)
}

func (h *HandlerManager) readMemory(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	payload := map[string]interface{}{
		"total_human":     config.BytesToHuman(memInfo.Total),
		"used_human":      config.BytesToHuman(memInfo.Used),
		"available_human": config.BytesToHuman(memInfo.Available),
		"usage_percent":   memInfo.UsedPercent,
	}

	return resourceText(request.Params.URI, payload)
}

func (h *HandlerManager) readDisk(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk partitions: %w", err)
	}

	disks := []map[string]interface{}{}
	for _, p := range partitions {
		if p.Fstype == "tmpfs" || p.Fstype == "devtmpfs" || p.Fstype == "squashfs" {
			continue
		}
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}
		disks = append(disks, map[string]interface{}{
			"mount_point":   p.Mountpoint,
			"total_human":   config.BytesToHuman(usage.Total),
			"used_human":    config.BytesToHuman(usage.Used),
			"free_human":    config.BytesToHuman(usage.Free),
			"usage_percent": usage.UsedPercent,
		})
	}

	return resourceText(request.Params.URI, map[string]interface{}{"disks": disks})
}

func (h *HandlerManager) readNetwork(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	payload := map[string]interface{}{}
	if snap, ok := h.monitor.Last(); ok {
		payload["throughput"] = snap.Net
	}

	// Fall back to a live sample if the monitor has not produced data yet.
	if len(payload) == 0 {
		snap, err := h.monitor.SampleNow(ctx)
		if err != nil {
			return nil, err
		}
		payload["throughput"] = snap.Net
	}

	return resourceText(request.Params.URI, payload)
}

func (h *HandlerManager) readProcesses(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	limit := 10
	if v, ok := request.Params.Arguments["limit"]; ok {
		if n, ok := v.(float64); ok && n > 0 {
			limit = int(n)
			if limit > 50 {
				limit = 50
			}
		}
	}

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{"limit": limit, "sort_by": "cpu"},
		},
	}
	result, err := h.HandleGetProcessList(ctx, req)
	if err != nil {
		return nil, err
	}

	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		return nil, fmt.Errorf("unexpected process list content type")
	}

	return []mcp.ResourceContents{mcp.TextResourceContents{
		URI:      request.Params.URI,
		MIMEType: "application/json",
		Text:     text.Text,
	}}, nil
}
