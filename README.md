# MCPBridge-Go

This is a simple app in Go that exposes locally installed MCPs (Model Context Protocol servers) over the network to AI agents running on different machines and servers. Instead of installing MCPs individually on each machine, you can centralize them on a server and expose them via HTTP/SSE.

[![Go Version](https://img.shields.io/badge/go-1.26.2+-blue.svg)](https://golang.org) [![License](https://img.shields.io/badge/license-MIT-green.svg)](#license)

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

Create a config.yml file:

```yaml
server:
  host: "0.0.0.0"

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
./mcpbridgego --config config.yml
```

That's it! Your MCPs are now accessible over the network.

## Configuration Guide

### Server Configuration

```yaml
server:
  host: "0.0.0.0"      # Bind address (0.0.0.0 for all interfaces)
```

### MCP Configuration

Each MCP entry in the `mcps` list requires:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Unique identifier for the MCP (used in logs and routing) |
| `port` | integer | Port to expose this MCP on |
| `command` | string | Command to execute (e.g., `node`, `python`, `bash`) |
| `args` | array | Command arguments (script path, options, etc.) |
| `env_file` | string | Path to `.env` file with environment variables for this MCP |

### Example: Multiple MCPs

```yaml
server:
  host: "0.0.0.0"

mcps:
  - name: "proxmox"
    port: 3000
    command: "node"
    args: ["/opt/mcps/mcp-proxmox/dist/index.js"]
    env_file: "/opt/mcps/mcp-proxmox/.env"
  
  - name: "kubernetes"
    port: 3001
    command: "python"
    args: ["/opt/mcps/k8s-mcp/main.py"]
    env_file: "/opt/mcps/k8s-mcp/.env"
  
  - name: "database"
    port: 3002
    command: "node"
    args: ["/opt/mcps/db-mcp/dist/index.js"]
    env_file: "/opt/mcps/db-mcp/.env"
```

## Usage Examples

### For Claude Desktop Users

Configure your Claude Desktop to use the exposed MCPs:

**On Linux/macOS** (`~/.config/Claude/claude_desktop_config.json`):

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

**On Windows** (`%APPDATA%\Claude\claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "proxmox": {
      "url": "sse://192.168.1.100:3000"
    }
  }
}
```

Then restart Claude Desktop and the MCPs will be available to your agent.

### For Custom AI Agents

Any AI agent that supports Server-Sent Events can connect to your MCPs:

```bash
# Using curl to test connectivity
curl -N http://192.168.1.100:3000
```

### Network Topology Example

```
┌─────────────────────────────────────┐
│       MCPBridge Server              │
│  (192.168.1.100)                    │
├─────────────────────────────────────┤
│ Port 3000: Proxmox MCP              │
│ Port 3001: Kubernetes MCP           │
│ Port 3002: Database MCP             │
└─────────────────────────────────────┘
          ▲
    ┌─────┼─────┬──────────┐
    │     │     │          │
┌───┴──┐┌─┴──┐┌─┴──┐   ┌──┴───┐
│Agent ││Agent││Agent│   │Agent │
│ Mac  ││Linux││Win  │   │Linux │
│      ││     ││     │   │      │
└──────┘└──────┘└──────┘   └──────┘
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

## Troubleshooting

### MCP won't start

1. **Check the configuration file path**
   ```bash
   ./mcpbridgego --config /path/to/config.yml
   ```

2. **Verify the command exists**
   ```bash
   which node
   which python
   ```

3. **Check script paths in config**
   ```bash
   ls -la /opt/mcps/mcp-proxmox/dist/index.js
   ```

4. **View logs**
   ```bash
   ./mcpbridgego --config config.yml 2>&1 | tee mcpbridge.log
   ```

### Connection refused from agent machines

1. **Check firewall rules**
   ```bash
   # Open ports (example for Linux)
   sudo ufw allow 3000:3002/tcp
   ```

2. **Verify MCPBridge is running**
   ```bash
   netstat -tlnp | grep mcpbridgego
   ```

3. **Test connectivity from client machine**
   ```bash
   curl -N http://SERVER_IP:3000
   ```

### Environment variables not loading

1. **Verify .env file path**
   ```bash
   cat /opt/mcps/mcp-proxmox/.env
   ```

2. **Check file permissions**
   ```bash
   ls -la /opt/mcps/mcp-proxmox/.env
   ```

## Contributing

We welcome contributions! Here's how to get started:

### Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/dipievil/mcpbridge-go.git
   cd mcpbridge-go
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Build the project**
   ```bash
   go build -o mcpbridgego ./cmd/mcpbridgego
   ```

### Making Changes

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**
   - Follow Go idioms and best practices
   - Keep commits atomic and well-documented
   - Add tests for new functionality

3. **Test your changes**
   ```bash
   go test ./...
   ```

4. **Submit a Pull Request**
   - Describe what your change does
   - Explain why it's needed
   - Reference any related issues

### Code Style

- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` for formatting
- Add comments for exported functions
- Keep functions focused and readable

### Reporting Issues

Found a bug? Please create an issue with:

- Clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Your environment (OS, Go version, MCPBridge version)
- Relevant logs or error messages

### Feature Requests

Have an idea? Open an issue with:

- Clear description of the feature
- Use case and benefits
- Potential implementation approach (if you have one)

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

- 📖 [Documentation](https://github.com/dipievil/mcpbridge-go)
- 🐛 [Report Issues](https://github.com/dipievil/mcpbridge-go/issues)
