package output

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"dipievil/mcpbridgego/internal/config"
)

const (
	ColorGreen  = "\033[32m"
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
)

// OutputConfig defines output options
type OutputConfig struct {
	Agent    string
	IsFile   bool
	FilePath string
}

// AgentTemplate contains MCP configuration templates for different agents
var AgentTemplates = map[string]map[string]interface{}{
	"claude": {
		"servers": map[string]interface{}{
			"context7": map[string]interface{}{
				"url": "http://localhost:3001",
			},
			"kubernetes": map[string]interface{}{
				"url": "sse://192.168.1.100:3001",
			},
			"database": map[string]interface{}{
				"url": "sse://192.168.1.100:3002",
			},
		},
	},
	"copilot": {
		"servers": map[string]interface{}{
			"context7": map[string]interface{}{
				"url": "http://localhost:3001",
			},
			"kubernetes": map[string]interface{}{
				"url": "sse://192.168.1.100:3001",
			},
			"database": map[string]interface{}{
				"url": "sse://192.168.1.100:3002",
			},
		},
	},
	"generic": {
		"servers": map[string]interface{}{
			"example_server": map[string]interface{}{
				"url": "http://localhost:3000",
			},
		},
	},
}

// ParseOutputConfig parses output configuration from CLI arguments
func ParseOutputConfig(agent string, isFile bool, filePath string) (OutputConfig, error) {
	agent = strings.ToLower(strings.TrimSpace(agent))

	if agent == "" {
		agent = "generic"
	}

	validAgents := []string{"claude", "copilot", "generic"}
	agentValid := false
	for _, valid := range validAgents {
		if agent == valid {
			agentValid = true
			break
		}
	}
	if !agentValid {
		return OutputConfig{}, fmt.Errorf("invalid agent: %s. Supported: claude, copilot, generic", agent)
	}

	cfg := OutputConfig{
		Agent:    agent,
		IsFile:   isFile,
		FilePath: filePath,
	}

	if isFile && filePath == "" {
		cfg.FilePath = "mcp.json"
	}

	return cfg, nil
}

// OutputMCPConfig outputs MCP configuration based on the provided options
func OutputMCPConfig(cfg OutputConfig) error {
	template, exists := AgentTemplates[cfg.Agent]
	if !exists {
		return fmt.Errorf("unknown agent type: %s. Supported agents: claude, copilot, generic", cfg.Agent)
	}

	jsonData, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %v", err)
	}

	if cfg.IsFile {
		return outputToFile(jsonData, cfg.FilePath)
	}
	outputToScreen(jsonData)
	return nil
}

// outputToScreen prints JSON to screen with green color
func outputToScreen(jsonData []byte) {
	fmt.Printf("%s%s%s\n", ColorGreen, string(jsonData), ColorReset)
}

// outputToFile writes JSON to file
func outputToFile(jsonData []byte, filePath string) error {

	if filePath == "" {
		filePath = "mcp.json"
	}

	dir := filepath.Dir(filePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creating directory: %v", err)
		}
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}

	fmt.Printf("%s✓ Configuration saved to: %s%s\n", ColorGreen, filePath, ColorReset)
	return nil
}

// PrintOutputUsage prints usage information for the output command
func PrintOutputUsage() {
	fmt.Printf(`%sUsage:%s
  # Output specific agent to screen (default behavior)
  mcpbridgego -o claude                              # Claude to screen
  mcpbridgego -o copilot                             # Copilot to screen
  mcpbridgego --output generic                       # Generic to screen
  
  # Output agent to file (default filename: mcp.json)
  mcpbridgego -o claude -f                           # Claude to mcp.json
  mcpbridgego --output copilot --file                # Copilot to mcp.json
  
  # Output to file with custom path
  mcpbridgego -o copilot -f ./agents/copilot-mcp.json
  mcpbridgego -f ./config/mcp-config.json            # Generic to custom path
  
  # Default behavior (generic to screen)
  mcpbridgego -o                                     # Same as: -o generic
  mcpbridgego                                        # If no other args, shows config

%sOptions:%s
  -o, --output <agent>   Agent type: claude, copilot, generic (default: generic)
  -f, --file [path]      Output to file (default path: mcp.json)
  -h, --help             Show this help message

%sExamples:%s
  # Output Claude MCP configuration to screen (green colored)
  mcpbridgego -o claude

  # Save GitHub Copilot configuration to file
  mcpbridgego -o copilot -f

  # Save generic configuration to custom path
  mcpbridgego -f ./config/agents/mcp-config.json

  # Multiple configurations
  mcpbridgego -o claude -f claude-mcp.json
  mcpbridgego -o copilot -f copilot-mcp.json
`, ColorBold, ColorReset, ColorBold, ColorReset, ColorBold, ColorReset)
}

// PrintMainHelp prints the main help message
func PrintMainHelp() {
	fmt.Println("MCPBridge - Model Context Protocol Bridge")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  mcpbridgego [options] [config_file]")
	fmt.Println()
	fmt.Println("Common options:")
	fmt.Println("  -start                   Start MCPBridge in background")
	fmt.Println("  -stop                    Stop the running MCPBridge")
	fmt.Println("  -h, --help               Show this help message")
	fmt.Println()
	fmt.Println("Output yml template:")
	fmt.Println("  -o, --output <agent>     Agent type: claude, copilot, generic (default: generic)")
	fmt.Println("  -f, --file [filename]    Output template to file (default: mcp.json)")
	fmt.Println()
	PrintOutputUsage()
}

// getLocalIPForServer returns the local IP address to use for MCP servers
// Tries to find a non-loopback IPv4 address
func getLocalIPForServer() string {
	defaultIP := "localhost"

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return defaultIP
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipv4 := ipnet.IP.To4(); ipv4 != nil {
				return ipv4.String()
			}
		}
	}

	return defaultIP
}

// generateDynamicMCPConfig generates MCP configuration from config file
// Returns JSON representation of MCP servers
func generateDynamicMCPConfig() (map[string]interface{}, error) {

	cfg, err := config.LoadConfig()

	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %v", err)
	}

	localIP := getLocalIPForServer()

	servers := make(map[string]interface{})
	for _, mcp := range cfg.MCPS {
		url := fmt.Sprintf("http://%s:%d", localIP, mcp.Port)
		servers[mcp.Name] = map[string]interface{}{
			"url": url,
		}
	}

	return map[string]interface{}{
		"servers": servers,
	}, nil
}

// DisplayAgentCfgInfo displays startup information when MCPBridge starts in background mode
// Shows the dynamic MCP configuration based on the config file
func DisplayAgentCfgInfo() {
	fmt.Println()
	fmt.Printf("%sMCP Configuration for your agents:%s\n", ColorBold, ColorBlue)
	fmt.Println()

	config, err := generateDynamicMCPConfig()
	if err != nil {
		fmt.Printf("%sWarning: Could not generate configuration: %v%s\n", ColorYellow, err, ColorReset)
		return
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Printf("%sWarning: Could not marshal configuration: %v%s\n", ColorYellow, err, ColorReset)
		return
	}

	outputToScreen(jsonData)
}
