// Package main is the entry point for the sysmetrics-mcp server.
package main

import (
	"flag"
	"fmt"
	"os"

	"sysmetrics-mcp/internal/config"
	"sysmetrics-mcp/internal/handlers"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	var cfg config.Config

	// Parse CLI flags
	flag.StringVar(&cfg.TempUnit, "temp-unit", "celsius", "Temperature unit: celsius, fahrenheit, or kelvin")
	flag.IntVar(&cfg.MaxProcesses, "max-processes", 10, "Maximum number of processes to list")
	flag.StringVar(&cfg.MountPointsStr, "mount-points", "", "Comma-separated mount points to monitor (empty = all)")
	flag.StringVar(&cfg.InterfacesStr, "interfaces", "", "Comma-separated interfaces to monitor (empty = all)")
	flag.BoolVar(&cfg.EnableGPU, "enable-gpu", true, "Attempt to read GPU metrics if available")
	flag.IntVar(&cfg.MonitorInterval, "monitor-interval", config.DefaultMonitorInterval, "Default sampling interval in seconds for monitoring")
	flag.Parse()

	// Validate and parse comma-separated lists
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Create MCP server
	s := server.NewMCPServer(
		"sysmetrics-mcp",
		"1.0.0",
	)

	// Enable sampling so the server can proactively push alerts to clients
	// that declare sampling support.
	s.EnableSampling()

	// Create handler manager and register tools, resources, and prompts
	hm := handlers.NewHandlerManager(&cfg)
	hm.RegisterTools(s)
	hm.RegisterMonitoringTools(s)
	hm.RegisterResources(s)
	hm.RegisterPrompts(s)

	// Start server via stdio
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
