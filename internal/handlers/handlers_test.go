package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"sysmetrics-mcp/internal/config"

	"github.com/mark3labs/mcp-go/mcp"
)

// Helper to check tool result
func checkToolResult(t *testing.T, res *mcp.CallToolResult, err error, expectedKeys []string) {
	t.Helper()
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if res.IsError {
		var msg string
		if content, ok := res.Content[0].(mcp.TextContent); ok {
			msg = content.Text
		} else {
			msg = fmt.Sprintf("Unknown content type: %T", res.Content[0])
		}
		t.Fatalf("Tool result is error: %v", msg)
	}

	var data map[string]interface{}
	textContent, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Result content not TextContent: %T", res.Content[0])
	}
	text := textContent.Text
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		t.Fatalf("Failed to decode JSON result: %v", err)
	}

	for _, key := range expectedKeys {
		if _, ok := data[key]; !ok {
			t.Errorf("Missing expected key in result: %s", key)
		}
	}
}

func TestHandleGetSystemInfo(t *testing.T) {
	h := NewHandlerManager(&config.Config{})
	req := mcp.CallToolRequest{}
	res, err := h.HandleGetSystemInfo(context.Background(), req)
	checkToolResult(t, res, err, []string{"hostname", "os", "platform", "uptime_seconds"})
}

func TestHandleGetCPUMetrics(t *testing.T) {
	// Setup config
	h := NewHandlerManager(&config.Config{TempUnit: "celsius"})

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}
	res, err := h.HandleGetCPUMetrics(context.Background(), req)
	checkToolResult(t, res, err, []string{"usage_percent", "per_cpu_percent", "core_count"})
}

func TestHandleGetMemoryMetrics(t *testing.T) {
	h := NewHandlerManager(&config.Config{})
	req := mcp.CallToolRequest{}
	res, err := h.HandleGetMemoryMetrics(context.Background(), req)
	checkToolResult(t, res, err, []string{"ram", "swap"})
}

func TestHandleGetDiskIOMetrics(t *testing.T) {
	h := NewHandlerManager(&config.Config{})
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}
	res, err := h.HandleGetDiskIOMetrics(context.Background(), req)
	checkToolResult(t, res, err, []string{"devices", "total"})
}

func TestHandleGetDiskIOMetricsWithFilter(t *testing.T) {
	h := NewHandlerManager(&config.Config{})
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"devices": "nonexistent_device",
			},
		},
	}
	res, err := h.HandleGetDiskIOMetrics(context.Background(), req)
	checkToolResult(t, res, err, []string{"devices", "total"})

	// Verify the filtered result returns 0 devices
	var data map[string]interface{}
	textContent := res.Content[0].(mcp.TextContent)
	if parseErr := json.Unmarshal([]byte(textContent.Text), &data); parseErr != nil {
		t.Fatalf("Failed to parse result: %v", parseErr)
	}
	if total := data["total"].(float64); total != 0 {
		t.Errorf("Expected 0 devices for nonexistent filter, got %v", total)
	}
}

func TestHandleGetSystemHealth(t *testing.T) {
	h := NewHandlerManager(&config.Config{})
	req := mcp.CallToolRequest{}
	res, err := h.HandleGetSystemHealth(context.Background(), req)
	checkToolResult(t, res, err, []string{"status", "cpu", "memory", "disk", "uptime", "hostname"})

	// Verify status is one of the expected values
	var data map[string]interface{}
	textContent := res.Content[0].(mcp.TextContent)
	if parseErr := json.Unmarshal([]byte(textContent.Text), &data); parseErr != nil {
		t.Fatalf("Failed to parse result: %v", parseErr)
	}
	status := data["status"].(string)
	if status != "healthy" && status != "warning" && status != "critical" {
		t.Errorf("Unexpected status: %s", status)
	}
}

func TestHandleGetDockerMetrics(t *testing.T) {
	h := NewHandlerManager(&config.Config{})
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}
	res, err := h.HandleGetDockerMetrics(context.Background(), req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	// Docker may not be available — either we get a tool error or valid JSON
	textContent, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Result content not TextContent: %T", res.Content[0])
	}
	if res.IsError {
		// Graceful degradation: Docker not available
		if !strings.Contains(textContent.Text, "Docker not available") {
			t.Errorf("Expected Docker unavailable message, got: %s", textContent.Text)
		}
		return
	}
	// Docker is available: verify JSON structure
	var data map[string]interface{}
	if parseErr := json.Unmarshal([]byte(textContent.Text), &data); parseErr != nil {
		t.Fatalf("Failed to parse result JSON: %v", parseErr)
	}
	if _, ok := data["containers"]; !ok {
		t.Error("Missing 'containers' key in Docker metrics result")
	}
	if _, ok := data["total"]; !ok {
		t.Error("Missing 'total' key in Docker metrics result")
	}
}

func TestHandleGetNetworkConnections(t *testing.T) {
	h := NewHandlerManager(&config.Config{})
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}
	res, err := h.HandleGetNetworkConnections(context.Background(), req)
	checkToolResult(t, res, err, []string{"connections", "total", "kind"})
}

func TestHandleGetNetworkConnectionsFiltered(t *testing.T) {
	h := NewHandlerManager(&config.Config{})
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"kind":   "tcp",
				"status": "LISTEN",
			},
		},
	}
	res, err := h.HandleGetNetworkConnections(context.Background(), req)
	checkToolResult(t, res, err, []string{"connections", "total", "kind", "status_filter"})
}

func TestHandleGetServiceStatus(t *testing.T) {
	h := NewHandlerManager(&config.Config{})
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"services": "ssh",
			},
		},
	}
	res, err := h.HandleGetServiceStatus(context.Background(), req)
	checkToolResult(t, res, err, []string{"services", "total"})
}

func TestHandleGetServiceStatusMissing(t *testing.T) {
	h := NewHandlerManager(&config.Config{})
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}
	res, err := h.HandleGetServiceStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	// Should return error result since services param is missing
	if !res.IsError {
		t.Error("Expected error result when services parameter is missing")
	}
}
