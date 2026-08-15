package handlers

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterPrompts registers MCP prompt templates.
func (h *HandlerManager) RegisterPrompts(s *server.MCPServer) {
	s.AddPrompt(mcp.NewPrompt("analyze_system_health",
		mcp.WithPromptDescription("Gather and summarize overall system health, flagging any warnings or critical issues."),
		mcp.WithArgument("detail", mcp.ArgumentDescription("Set to 'full' for a detailed report")),
	), h.promptAnalyzeHealth)

	s.AddPrompt(mcp.NewPrompt("diagnose_performance_issue",
		mcp.WithPromptDescription("Diagnose a reported performance problem by checking CPU, memory, disk, network, and top processes."),
		mcp.WithArgument("symptom", mcp.ArgumentDescription("Description of the performance issue"), mcp.RequiredArgument()),
	), h.promptDiagnosePerformance)
}

func (h *HandlerManager) promptAnalyzeHealth(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	messages := []mcp.PromptMessage{
		{
			Role: mcp.RoleUser,
			Content: mcp.TextContent{
				Type: "text",
				Text: "Analyze the health of this system. Start by calling get_system_health. " +
					"If the status is not healthy, call get_cpu_metrics, get_memory_metrics, and get_alerts " +
					"to gather details, then summarize the issues and suggest corrective actions.",
			},
		},
	}
	return mcp.NewGetPromptResult("System health analysis", messages), nil
}

func (h *HandlerManager) promptDiagnosePerformance(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	symptom := ""
	if v, ok := request.Params.Arguments["symptom"]; ok {
		symptom = v
	}

	messages := []mcp.PromptMessage{
		{
			Role: mcp.RoleUser,
			Content: mcp.TextContent{
				Type: "text",
				Text: "Diagnose this performance issue: " + symptom + ". " +
					"Gather get_system_health, get_cpu_metrics, get_memory_metrics, get_disk_metrics, " +
					"get_process_list (sorted by cpu and memory), and get_alerts. " +
					"Then provide a concise diagnosis and prioritized recommendations.",
			},
		},
	}
	return mcp.NewGetPromptResult("Performance diagnosis", messages), nil
}
