# AI Agent Guidelines for SysMetrics MCP

This document provides essential instructions for autonomous coding agents (like Copilot, Cursor, OpenCode, Gemini, etc.) operating within the **SysMetrics MCP** repository.

## 1. Project Overview
- **Name**: SysMetrics MCP
- **Type**: Model Context Protocol (MCP) Server for Linux system metrics.
- **Language**: Go 1.25.6+
- **Architecture**: `cmd/sysmetrics-mcp` (entrypoint), `internal/config` (CLI/config parsing), `internal/handlers` (core metrics logic).
- **Key Libraries**: `github.com/mark3labs/mcp-go` (MCP Framework), `github.com/shirou/gopsutil/v3` (System Metrics).

## 2. Build, Lint, and Test Commands

We use a `Makefile` for standard development tasks. Always prefer these targets to ensure consistency.

### Common Commands
- **Build**: `make build`
  - Compiles the project to `bin/sysmetrics-mcp` with `CGO_ENABLED=0`.
- **Run Locally**: `go run . [flags]`
- **Format Code**: `make fmt` (runs `go fmt ./...`)
- **Lint Code**: `make lint` (runs `golangci-lint run` and `go vet ./...`)
- **Update Dependencies**: `make deps` (runs `go mod download` and `go mod tidy`)

### Testing Commands
Tests are critical. Always verify your changes before finalizing them.

- **Run all tests**:
  ```bash
  make test
  # Under the hood: CGO_ENABLED=0 go test -v ./...
  ```
- **Run tests in a specific package**:
  ```bash
  CGO_ENABLED=0 go test -v ./internal/handlers
  ```
- **Run a single test** (highly recommended for TDD/debugging):
  ```bash
  CGO_ENABLED=0 go test -v -run ^TestFunctionName$ ./internal/handlers
  ```
- **Run tests with coverage**:
  ```bash
  CGO_ENABLED=0 go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out
  ```

## 3. Code Style & Architecture Guidelines

Adhere strictly to standard Go idioms and the existing patterns in this repository.

### Formatting & Imports
- **Formatter**: Use standard `go fmt`. Run `make fmt` before committing.
- **Imports**: Group imports into three distinct blocks separated by a blank line:
  1. Standard library packages.
  2. Third-party packages (e.g., `github.com/shirou/gopsutil/v3/...`).
  3. Internal project packages (`sysmetrics-mcp/internal/...`).
- **Linter Strictness**: The project uses `golangci-lint` with strict rules (e.g., `revive`, `gosec`, `gocritic`). Your code *must* pass `make lint` cleanly without adding `#nosec` or `nolint` pragmas unless absolutely necessary and documented.

### Naming Conventions
- **General**: Use `camelCase` for variables and `PascalCase` for exported identifiers.
- **Acronyms**: Keep initialisms uppercase (e.g., `HTTPClient`, `UserID`, not `HttpClient` or `UserId`).
- **Interfaces**: Interface names should typically end in `-er` (e.g., `MetricFetcher`, `ConfigParser`).
- **Packages**: Package names should be short, lowercase, and avoid snake_case or hyphens. Do not use generic names like `util` or `common`.

### Error Handling (CRITICAL)
- **Do not panic**: The server must run continuously. Never use `panic()` or `log.Fatal()` inside handlers or internal packages.
- **MCP Errors**: Handlers must return structured error messages via `mcp.NewToolResultError` instead of crashing the server or returning raw Go errors to the MCP transport.
- **Contextual Errors**: When returning errors internally, wrap them to provide context:
  ```go
  return fmt.Errorf("failed to fetch disk metrics for %s: %w", mountPoint, err)
  ```
- **Graceful Degradation**: If a specific metric fails to load (e.g., Raspberry Pi GPU temp on a standard Linux machine), log a warning or return `N/A` instead of failing the entire tool request.

### Types and Data Structures
- Favor strong typing. Avoid `interface{}` (or `any`) unless implementing generic data structures.
- Use pointers for structs only when they need to be mutated or when passing large configurations. Otherwise, pass by value.
- Prefer explicit struct initialization over positional arguments.

### Testing Conventions
- **Table-Driven Tests**: Use table-driven tests for complex logic (refer to existing tests in `config_test.go`).
- **Mocking**: Handlers are unit tested by mocking or calling them directly with a context (`handlers_test.go`). Focus on testing the business logic independently of the MCP transport layer.
- **Asserts**: Use standard library `testing` package patterns. Avoid heavy third-party assertion libraries unless already present in the `go.mod`.

### Project-Specific Rules
- **Platform Awareness**: The code detects Raspberry Pi hardware to provide additional metrics (GPU temp, throttling). Keep fallback paths clean for generic Linux environments.
- **Tools Addition**: When adding a new MCP tool, ensure it is properly registered in `cmd/sysmetrics-mcp/main.go` and its handler is implemented in `internal/handlers` with appropriate error handling and JSON serialization.

## 4. Agent Interaction Protocol

When acting on user requests in this repository:
1. **Understand Context**: Read `Makefile`, `.golangci.yml`, and `GEMINI.md` to understand constraints. Use `grep` and `glob` to locate relevant handler files before writing code.
2. **Plan**: Write out a concise plan including the exact files to be modified and tests to be written.
3. **Execute**: Modify the code using absolute file paths.
4. **Self-Verify**: ALWAYS run `make fmt`, `make lint`, and the specific `go test -run ...` command for the code you touched. Do not finalize until these pass.
