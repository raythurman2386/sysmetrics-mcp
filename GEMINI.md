# SysMetrics MCP Server - Project Context

A lightweight [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server written in Go that provides comprehensive Linux system metrics to AI models. It is optimized for both generic Linux distributions and Raspberry Pi systems.

## Project Overview

- **Core Functionality**: Exposes system metrics (CPU, Memory, Disk, Disk I/O, Network, Network Connections, Processes, Thermal, Docker, System Health, Service Status) as MCP tools.
- **Main Technologies**:
  - **Language**: Go 1.25.6+
  - **MCP Framework**: `github.com/mark3labs/mcp-go`
  - **Metrics Library**: `github.com/shirou/gopsutil/v3`
- **License**: MIT
- **Architecture**:
  - `cmd/sysmetrics-mcp/main.go`: Entry point and server lifecycle management.
  - `internal/config/config.go`: CLI flag parsing and validation.
  - `internal/handlers/handlers.go`: Core logic for fetching system metrics.
  - `bin/sysmetrics-mcp`: The compiled binary (ignored by git).

## Building and Running

The project includes a `Makefile` for common development tasks.

### Key Commands

- **Build**: `make build` (Produces the `sysmetrics-mcp` binary)
- **Test**: `make test` (Runs all unit tests with verbose output)
- **Lint**: `make lint` (Runs `go vet` and checks code quality)
- **Install**: `make install` (Builds and copies the binary to `/usr/local/bin`)
- **Clean**: `make clean` (Removes compiled binaries and temporary files)
- **Dependencies**: `make deps` (Downloads and tidies Go modules)

### Development Loop

To run the server locally during development (using stdio transport):
```bash
go run . [flags]
```

## Configuration

The server is configured via CLI flags, which are handled in `config.go`.

| Flag | Default | Description |
|------|---------|-------------|
| `--temp-unit` | `celsius` | `celsius`, `fahrenheit`, or `kelvin` |
| `--max-processes` | `10` | Limit for process list (1-50) |
| `--mount-points` | `""` | Comma-separated list of mount points to monitor |
| `--interfaces` | `""` | Comma-separated list of network interfaces to monitor |
| `--enable-gpu` | `true` | Enable Raspberry Pi GPU metrics via `vcgencmd` |

## Development Conventions

- **Go Standards**: Adheres to standard Go idioms and project structure.
- **Error Handling**: Handlers return structured error messages via `mcp.NewToolResultError` instead of crashing the server.
- **Testing**:
  - Uses table-driven tests for logic (see `config_test.go`).
  - Handlers are unit tested by mocking/calling them directly with context (see `handlers_test.go`).
- **Raspberry Pi Specialization**: The code detects Raspberry Pi hardware to provide additional metrics (GPU temp, throttling status) while falling back gracefully on generic Linux systems.
- **Linting**: Pre-configured for `golangci-lint` (see `.golangci.yml`), including linters like `revive`, `gosec`, and `gocritic`.

## MCP Tools

The following tools are available to the AI:

1.  `get_system_info`: Hostname, OS, kernel, uptime.
2.  `get_cpu_metrics`: Usage, per-core load, temperature.
3.  `get_memory_metrics`: Virtual memory and Swap usage.
4.  `get_disk_metrics`: Disk usage per mount point.
5.  `get_network_metrics`: Interface statistics and IP addresses.
6.  `get_process_list`: Top processes by CPU/Memory.
7.  `get_thermal_status`: Advanced thermal and throttling info (Pi-optimized).
8.  `get_disk_io_metrics`: Disk I/O throughput, IOPS, and read/write times.
9.  `get_system_health`: Aggregated health dashboard (CPU, memory, disk, uptime, status).
10. `get_docker_metrics`: Docker container CPU and memory metrics via cgroups.
11. `get_network_connections`: Active TCP/UDP connections with PID and status.
12. `get_service_status`: Systemd service health via `systemctl show`.
