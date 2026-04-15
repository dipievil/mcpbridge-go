package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	jsonData, err := json.MarshalIndent(template, "", "\t")
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
	fmt.Printf("%s%s%s%s\n", ColorBold, ColorGreen, string(jsonData), ColorReset)
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

// PrintStartupInfo displays startup information when MCPBridge starts in background mode
// Shows the generic MCP configuration and instructions for configuration export
func PrintStartupInfo() {
	fmt.Println()
	fmt.Printf("%s════════════════════════════════════════════════════════════════%s\n", ColorBold, ColorReset)
	fmt.Printf("%s  MCP Bridge Server is running in %s background mode%s\n", ColorBlue, ColorYellow, ColorReset)
	fmt.Printf("%s════════════════════════════════════════════════════════════════%s\n", ColorBold, ColorReset)
	fmt.Println()

	fmt.Printf("%sMCP Configuration for your agents:%s\n", ColorBold, ColorReset)
	fmt.Println()

	outputCfg := OutputConfig{
		Agent:  "generic",
		IsFile: false,
	}
	if err := OutputMCPConfig(outputCfg); err != nil {
		fmt.Printf("%sWarning: Could not display configuration: %v%s\n", ColorYellow, err, ColorReset)
	}
}
