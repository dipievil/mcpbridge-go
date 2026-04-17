# MCPBridge-Go

This is a simple app in Go that exposes locally installed MCPs (Model Context Protocol servers) over the network to AI agents running on different machines and servers. Instead of installing MCPs individually on each machine, you can centralize them on a server and expose them via HTTP/SSE.

[![Go Version](https://img.shields.io/badge/go-1.26.2+-blue.svg)](https://golang.org) [![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE) [![Build Status](https://github.com/dipievil/mcpbridge-go/actions/workflows/build.yml/badge.svg)](https://github.com/dipievil/mcpbridge-go)

## Why MCPBridge-Go?

- 🌐 **Centralized MCPs**: Install MCPs once on one server instead of managing them across multiple machines
- 🤖 **Multi-Agent Support**: Multiple AI agents on different machines can connect to the same MCPs
- 🚀 **Easy Deployment**: Simple configuration, pre-built binaries ready to use
- 🔧 **Flexible**: Support for any MCP that runs via command line
- 📦 **Zero Dependencies**: Lightweight, self-contained executable

## Quick Start

### 1. Download

Get the latest release for your platform:

```bash

# Download the binary
wget https://github.com/dipievil/mcpbridge-go/releases/download/v1.0.0/mcpbridgego

# Make executable
chmod +x mcpbridgego
```

### 2. Configure

Create a `config.yaml` file:

```yaml
commands:
  - name: "node"
    path: "/usr/bin/node"

mcps:
  - name: "proxmox"
    port: 3000
    command: "node"
    args: ["/opt/mcps/mcp-proxmox/dist/index.js"]
    env_file: "/opt/mcps/mcp-proxmox/.env"
  
  - name: "loki"
    port: 3001
    command: "node"
    args: ["/opt/mcps/simple-loki-mcp/dist/index.js"]
    env_file: "/opt/mcps/simple-loki-mcp/.env"
```

### 3. Run

```bash
# Run in foreground (default, useful for development)
./mcpbridgego

# Or run in background (daemon mode)
./mcpbridgego --start

# Stop the background process
./mcpbridgego --stop
```

That's it! Your MCPs are now accessible over the network.

## Running Modes

MCPBridge supports two execution modes:

### Foreground Mode (Default)

```bash
./mcpbridgego
```

Runs in the current terminal. Requires `config.yaml` in the current directory. Useful for:

- Development and debugging
- Monitoring logs in real-time
- Running in containers (Docker, systemd, etc.)

### Daemon Mode

```bash
# Start in background
./mcpbridgego --start

# Stop the background process
./mcpbridgego --stop

# Check if running
ps aux | grep mcpbridgego
```

When starting in daemon mode, MCPBridge displays:

- **Generic MCP Configuration** - Pre-formatted JSON showing the basic server structure
- **Quick Reference Guide** - Common commands for exporting configurations
- **Usage Instructions** - How to export configs for Claude, GitHub Copilot, or generic agents
- **Process Information** - Background process PID

This helps you immediately understand how to configure your AI agents without needing to refer to documentation.

Runs in the background with:

- PID file stored (default: `/var/run/mcpbridgego.pid` or system temp directory)
- Graceful shutdown support  
- Useful for production deployments

## Features

### Core Features

- **Multi-MCP Support** - Run multiple MCPs on different ports simultaneously
- **Centralized Configuration** - Single YAML file to manage all MCPs
- **Environment Management** - Support for `.env` files and inline variables
- **Working Directory Support** - Specify working directory for each MCP process
- **JSON-RPC 2.0 Protocol** - Standard MCP communication protocol
- **Server-Sent Events** - Streaming responses for real-time updates
- **Health Checks** - Monitor MCP status and process information

### Operational Features

- **Daemon Mode** - Run in background with PID file management
- **Graceful Shutdown** - SIGTERM/SIGINT handling for clean process termination
- **CORS Support** - Cross-Origin Resource Sharing enabled by default
- **Pre-Startup Validation** - Checks config, ports, env files, and commands before starting
- **Stderr Logging** - Captures and logs MCP process error output
- **Request Timeout** - 30-second timeout for RPC calls to prevent hanging
- **Port Management** - Each MCP runs on its own port

## MCP Configuration Export

Export MCP configurations for different AI agents with a single command:

```bash
# Display Claude MCP configuration to screen (formatted with green color)
./mcpbridgego -o claude

# Save GitHub Copilot configuration to file (default: mcp.json)
./mcpbridgego -o copilot -f

# Export generic configuration to custom path
./mcpbridgego -f ./agents/mcp-config.json
```

### Supported Agents

- **claude** - Configuration for Claude agents
- **copilot** - Configuration for GitHub Copilot
- **generic** - Generic MCP server configuration (default)

### Output Options

- **screen** - Display configuration in terminal (formatted with green color) - **default**
- **-f, --file** - Save to JSON file (optional path, default filename: `mcp.json`)

### Quick Reference

| Command | Output | File |
| ------- | ------ | ---- |
| `-o claude` | Claude to screen | No |
| `-o copilot -f` | Copilot to file | `mcp.json` |
| `-f ./path/file.json` | Generic to file | `./path/file.json` |
| `-o` | Generic to screen | No |

## Configuration Reference

### Commands Registry

The optional `commands` section registers command aliases with their full paths:

| Field | Type | Description |
| ----- | ---- | ----------- |
| `name` | string | Command alias (e.g., `node`, `npx`) |
| `path` | string | Full path to the executable |

### MCP Configuration

The application reads configuration from `config.yaml` in the current directory.

Each MCP entry in the `mcps` list requires:

| Field | Type | Description |
| ----- | ---- | ----------- |
| `name` | string | Unique identifier for the MCP (used in logs and routing) |
| `port` | integer | Port to expose this MCP on |
| `command` | string | Command to execute (e.g., `node`, `python`, `bash`) |
| `args` | array | Command arguments (script path, options, etc.) |
| `env_file` | string | *(Optional)* Path to `.env` file with environment variables for this MCP |
| `env_vars` | object | *(Optional)* Environment variables defined directly in config (merged with `env_file`) |
| `dir` | string | *(Optional)* Working directory for the MCP process |

### Example: Multiple MCPs with Advanced Configuration

```yaml
commands:
  - name: "node"
    path: "/usr/bin/node"
  - name: "python"
    path: "/usr/bin/python3"

mcps:
  - name: "proxmox"
    port: 3000
    command: "node"
    args: ["/opt/mcps/mcp-proxmox/dist/index.js"]
    env_file: "/opt/mcps/mcp-proxmox/.env"
    env_vars:
      CUSTOM_VAR: "value"
  
  - name: "kubernetes"
    port: 3001
    command: "python"
    args: ["/opt/mcps/k8s-mcp/main.py"]
    env_file: "/opt/mcps/k8s-mcp/.env"
    dir: "/opt/mcps/k8s-mcp"  # Working directory
  
  - name: "database"
    port: 3002
    command: "node"
    args: ["/opt/mcps/db-mcp/dist/index.js"]
    env_vars:
      DB_HOST: "localhost"
      DB_PORT: "5432"
```

## Usage Examples

### Consuming the MCPBridge API

Each MCP is exposed with three HTTP endpoints:

- `POST /rpc`: JSON-RPC method calls
- `GET /sse`: Server-Sent Events stream
- `GET /health`: Health check
- `GET /`: Service info with available endpoints (JSON)

**CORS:** All endpoints respond to requests from any origin (Access-Control-Allow-Origin: *)

#### 1. JSON-RPC Endpoint (`/rpc`)

For standard JSON-RPC 2.0 method calls:

```bash
# Request
curl -X POST http://SERVER_IP:3000/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "your_method",
    "params": {"key": "value"},
    "id": 1
  }'

# Response
{
  "jsonrpc": "2.0",
  "result": {...},
  "id": 1
}
```

#### 2. Server-Sent Events Endpoint (`/sse`)

For streaming responses:

```bash
# Connect to SSE stream
curl -N http://SERVER_IP:3000/sse

# You'll receive JSON-RPC messages as Server-Sent Events
```

#### 3. Health Check Endpoint (`/health`)

For monitoring MCP status:

```bash
curl http://SERVER_IP:3000/health

# Response
{
  "status": "ok",
  "mcp": "proxmox",
  "pid": 12345,
  "port": 3000
}
```

#### 4. Root Endpoint (`/`)

The root endpoint serves as an alias for the SSE endpoint:

```bash
# Connect to SSE stream via root
curl -N http://SERVER_IP:3000/
```

### Exporting Configurations for AI Agents

Output specific agent to terminal (default behavior)

```bash
./mcpbridgego -o claude                                 # Claude to screen
./mcpbridgego -o copilot                                # Copilot to screen
./mcpbridgego --output generic                          # Generic to screen
./mcpbridgego -o                                        # Same as: generic to screen
```

Output agent to file (default filename: mcp.json)

```bash
 ./mcpbridgego -o claude -f                              # Claude to mcp.json
./mcpbridgego --output copilot --file                   # Copilot to mcp.json
```

Output to file with custom path

```bash
./mcpbridgego -o copilot -f ./agents/copilot-mcp.json   # Copilot with custom filename
./mcpbridgego -f ./config/mcp-config.json               # Generic to custom path
```

Save multiple configurations

```bash
./mcpbridgego -o claude -f claude-mcp.json
./mcpbridgego -o copilot -f copilot-mcp.json
./mcpbridgego -f ./generic/mcp.json                     # Generic to specific path
```

### For Claude Desktop Users

Configure your Claude Desktop to use the exposed MCPs:

```json
{
  "mcpServers": {
    "proxmox": {
      "url": "sse://192.168.1.100:3000"
    },
    "kubernetes": {
      "url": "sse://192.168.1.100:3001"
    },
    "database": {
      "url": "sse://192.168.1.100:3002"
    }
  }
}
```

Then restart Claude Desktop and the MCPs will be available to your agent.

### Custom AI Agents

Any AI agent that supports JSON-RPC 2.0 or Server-Sent Events can connect to your MCPs:

```bash
# Test JSON-RPC connectivity
curl -X POST http://SERVER_IP:3000/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "method": "test", "id": 1}'

# Test SSE connectivity
curl -N http://SERVER_IP:3000/sse
```

### Topology Example

```plaintext
┌───────────────────────────────────┐
│          MCPBridge Server         │
│            (192.168.1.1)          │
├───────────────────────────────────┤
│ Port 3000: Proxmox MCP            │
│ Port 3001: Kubernetes MCP         │
│ Port 3002: Database MCP           │
└───────────────────────────────────┘
                 ▲
    ┌────────┬────────┬────────┐
    │        │        │        │
┌───┴────┐┌──┴───┐┌───┴───┐┌───┴────┐
│ Claude ││VsCode││Copilot││ Gemini │
│        ││      ││  CLI  ││        │
└────────┘└──────┘└───────┘└────────┘
```

## Installation from Source

For developers who want to build from source:

### Requirements

- Go 1.26.2 or higher
- Make (optional)

### Build

```bash
# Clone the repository
git clone https://github.com/dipievil/mcpbridge-go.git
cd mcpbridge-go

# Build
go build -o mcpbridgego ./cmd/mcpbridgego

# Or with make
make build
```

### Testing

Run tests locally during development:

```bash
# Run all tests with coverage
make test

# Or with go directly
go test -v -race -coverpkg=./... ./...

# Run specific tests
go test -v -run TestBridgeInitialization ./...
```

### End-to-End (E2E) Testing

Test the complete MCPBridge functionality with mock MCP servers:

```bash
# Run all E2E tests (compiles bridge and mock servers automatically)
make e2e

# Or with go directly
go test -v -tags=e2e ./tests

# Run all tests including E2E
make test-all
```

**What E2E tests do:**

- Creates a lightweight mock MCP server that responds to JSON-RPC calls
- Starts MCPBridge in foreground mode
- Tests multiple communication patterns and 4 essential MCP protocol methods for each:
  1. **ping** - Connection verification
  2. **initialize** - Protocol initialization with server info
  3. **resources/list** - Resource discovery
  4. **tools/list** - Tool discovery
- Validates responses and cleans up resources

**E2E Test Structure:**

```plaintext
tests/
├── e2e_test.go              # HTTP/JSON-RPC E2E tests (4 test cases)
├── e2e_sse_test.go          # Server-Sent Events streaming tests (4 test cases)
├── mcp_mock_server/
│   └── main.go              # Lightweight mock MCP server implementation
└── .env                      # Environment variables for tests
```

**Communication Patterns Tested:**

1. **HTTP/JSON-RPC** (`e2e_test.go`):
   - POST requests with JSON-RPC messages to `/rpc` endpoint
   - Synchronous request-response pattern
   - Perfect for simple command/query operations

2. **HTTP/SSE** (`e2e_sse_test.go`):
   - POST requests with JSON-RPC messages to `/sse` endpoint
   - Server-Sent Events streaming format (`data: {json}\n\n`)
   - Supports bidirectional communication over HTTP
   - Ideal for long-lived connections and real-time updates

### Code Quality

```bash
# Format code
make fmt

# Run linter (go vet)
make lint

# Or all checks at once
go fmt ./... && go vet ./... && go test ./...
```

## Troubleshooting

### Pre-Startup Validation

Before accepting requests, MCPBridge validates:

1. **Config validity** - YAML file is properly formatted
2. **Env files** - Each `.env` file referenced in config exists
3. **Ports** - All configured ports are available
4. **Commands** - All commands specified exist in PATH

If validation fails, a clear error message is shown with the issue.

### MCP won't start

1. **Check the configuration file syntax**

   ```bash
   # Verify YAML is valid - requires config.yaml in the current directory
   ./mcpbridgego
   # MCPBridge will validate before starting
   ```

2. **Check if required command exists**

   ```bash
   which node
   which python
   ```

3. **Verify env file is accessible**

   ```bash
   ls -la /opt/mcps/mcp-proxmox/.env
   ```

4. **Check MCP stderr output**

   ```bash
   # When running in foreground, stderr from MCPs is logged with [mcp_name stderr] prefix
   ./mcpbridgego 2>&1 | grep stderr
   ```

5. **Verify working directory exists (if using `dir` config)**

   ```bash
   ls -la /path/to/mcp/dir
   ```

### Connection refused from agent machines

1. **Check if MCPBridge is running**

   ```bash
   # Check foreground process
   ps aux | grep mcpbridgego
   
   # Check daemon mode
   netstat -tlnp | grep mcpbridgego
   ```

2. **Verify firewall rules**

   ```bash
   # Example: Open ports 3000-3002 on Linux
   sudo ufw allow 3000:3002/tcp
   
   # Test connectivity from another machine
   curl http://SERVER_IP:3000/health
   ```

3. **Check port availability**

   ```bash
   netstat -tlnp | grep 3000
   ```

4. **Verify CORS is enabled** (enabled by default)
   - MCPBridge automatically adds `Access-Control-Allow-Origin: *` headers
   - Supports both `/rpc` (POST) and `/sse` (GET)

### Environment variables not loading

1. **Check env file path and permissions**

   ```bash
   # Verify file exists and is readable
   ls -la /opt/mcps/mcp-proxmox/.env
   cat /opt/mcps/mcp-proxmox/.env
   ```

2. **Verify env_file is in config** (optional field)

   ```yaml
   mcps:
     - name: "proxmox"
       env_file: "/path/to/.env"  # If left empty, file loading is skipped
       env_vars:                   # Can also define vars directly
         KEY: "value"
   ```

3. **Understand variable priority**

   - `env_file` is loaded first
   - `env_vars` in config override `env_file` values
   - Both are added to MCP's environment

4. **Confirm variables are set**

   ```bash
   # Run in foreground to see logging
   ./mcpbridgego 2>&1 | grep "Environment variables"
   ```

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines on:

- How to set up your development environment
- How to make and commit changes
- How to submit pull requests
- Code style and testing requirements

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

- 📖 [Documentation](https://github.com/dipievil/mcpbridge-go)
- 🐛 [Report Issues](https://github.com/dipievil/mcpbridge-go/issues)
