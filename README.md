# SysMetrics MCP Server

A lightweight MCP (Model Context Protocol) server that exposes Linux system metrics through MCP tools. Works on any Linux system including Raspberry Pi.

## Features

- **12 MCP Tools**: System info, CPU, memory, disk, disk I/O, network, network connections, processes, thermal, Docker, system health, and service status
- **Configurable**: CLI arguments for temperature units, process limits, mount points, and interfaces
- **Cross-Platform**: Works on any Linux system (enhanced metrics for Raspberry Pi)
- **AI-Ready**: Designed for integration with Claude Desktop, Cursor, or any MCP client

## Installation

### Prerequisites

- Go 1.25.6 or higher
- Linux system (tested on Ubuntu, Debian, Raspberry Pi OS)

### Build from Source

The project uses a `Makefile` for common tasks.

```bash
git clone <repository>
cd sysmetrics-mcp
make build
```

The compiled binary will be located in `bin/sysmetrics-mcp`.

### Install to PATH

```bash
# Option 1: System-wide (installs to /usr/local/bin)
sudo make install

# Option 2: User-local
mkdir -p ~/.local/bin
cp bin/sysmetrics-mcp ~/.local/bin/
# Add to PATH if not already: export PATH="$HOME/.local/bin:$PATH"
```

### Verify Installation

```bash
sysmetrics-mcp --help
```

## Configuration

### Local AI Agents (Gemini CLI / Personal Agents)

Add to your agent's configuration file:

```json
{
  "sysmetrics": {
    "type": "stdio",
    "command": "sysmetrics-mcp",
    "args": [
      "--temp-unit", "celsius",
      "--max-processes", "10"
    ]
  }
}
```

### Available CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--temp-unit` | `celsius` | Temperature unit: `celsius`, `fahrenheit`, or `kelvin` |
| `--max-processes` | `10` | Default maximum processes to list (1-50) |
| `--mount-points` | `""` | Comma-separated mount points (empty = all) |
| `--interfaces` | `""` | Comma-separated interfaces (empty = all, excludes `lo`) |
| `--enable-gpu` | `true` | Attempt to read GPU metrics (Raspberry Pi only) |

## MCP Tools

### `get_system_info`
Returns system information including hostname, OS, uptime, and platform details.

### `get_cpu_metrics`
Returns CPU usage, temperature, core count, and load average.

**Optional Arguments:**
- `temp_unit`: Override temperature unit

### `get_memory_metrics`
Returns RAM and swap usage statistics with both bytes and human-readable formats.

### `get_disk_metrics`
Returns disk usage for all or specified mount points.

**Optional Arguments:**
- `mount_points`: Comma-separated mount points to check
- `human_readable`: Include human-readable sizes (default: true)

### `get_network_metrics`
Returns network interface statistics including bytes sent/received and IP addresses.

**Optional Arguments:**
- `interfaces`: Comma-separated interface names to check

### `get_process_list`
Returns list of running processes sorted by resource usage.

**Optional Arguments:**
- `limit`: Maximum number of processes (1-50)
- `sort_by`: Sort by `cpu`, `memory`, or `pid` (default: `cpu`)

### `get_thermal_status`
Returns thermal status including CPU/GPU temperatures and throttling information (Raspberry Pi).

**Optional Arguments:**
- `temp_unit`: Override temperature unit

### `get_disk_io_metrics`
Returns disk I/O statistics including read/write throughput, IOPS, and I/O time per device.

**Optional Arguments:**
- `devices`: Comma-separated device names to check (e.g. `sda,nvme0n1`)

### `get_system_health`
Returns an aggregated health dashboard with CPU, memory, disk, and uptime. Includes an overall status of `healthy`, `warning`, or `critical` based on resource thresholds.

### `get_docker_metrics`
Returns Docker container metrics including CPU and memory usage via cgroups. Returns an empty list gracefully if Docker is not available.

**Optional Arguments:**
- `container_id`: Filter to a specific container by ID or name

### `get_network_connections`
Returns active TCP/UDP network connections with local/remote addresses, status, and owning PID.

**Optional Arguments:**
- `kind`: Connection type filter (`tcp`, `udp`, or `all`; default: `all`)
- `status`: Filter by connection status (e.g. `LISTEN`, `ESTABLISHED`)

### `get_service_status`
Returns systemd service health information via `systemctl show`.

**Required Arguments:**
- `services`: Comma-separated list of service names to check

## Example Usage

Once configured, you can ask your AI assistant:

- "What's my CPU temperature?"
- "Show me disk usage for / and /home"
- "List the top 5 processes by memory usage"
- "What's my network usage on eth0?"
- "Check if my Raspberry Pi is throttling"
- "What's the overall health of my system?"
- "Show me active TCP connections in LISTEN state"
- "Check if the SSH and Docker services are running"
- "What are the disk I/O stats for my drives?"
- "How much CPU and memory are my Docker containers using?"

## Raspberry Pi Enhancements

On Raspberry Pi systems, the server provides additional metrics:

- **CPU Temperature**: Reads from `/sys/class/thermal/thermal_zone0/temp`
- **GPU Temperature**: Uses `vcgencmd measure_temp`
- **Throttling Status**: Uses `vcgencmd get_throttled` to detect:
  - Under-voltage conditions
  - Frequency capping
  - Thermal throttling
  - Soft temperature limits

On non-Pi systems, these metrics return `"not_available"` gracefully.

## Development

Use the included `Makefile` for development tasks:

```bash
# Run tests
make test

# Run linter (go vet)
make lint

# Clean build artifacts
make clean

# Download and tidy dependencies
make deps
```

## Requirements

- Go 1.25.6+
- Linux system
- For Pi features: Raspberry Pi OS with `vcgencmd` available

## License

MIT
