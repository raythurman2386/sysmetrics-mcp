package handlers

import (
	"context"
	"encoding/json"
	"fmt"
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
