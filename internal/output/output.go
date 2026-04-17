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

// agentURLScheme maps agent names to their preferred URL scheme
var agentURLScheme = map[string]string{
	"claude":  "sse",
	"copilot": "http",
	"generic": "http",
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
	template, err := generateAgentConfig(cfg.Agent)
	if err != nil {
		return err
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

func PrintVersion(currentVersion string) {
	fmt.Printf("MCPBridge Version: %s\n", currentVersion)
}

// generateAgentConfig builds an agent-specific MCP config dynamically from config.yaml
func generateAgentConfig(agent string) (map[string]interface{}, error) {
	appCfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %v", err)
	}

	localIP := getLocalIPForServer()
	scheme := agentURLScheme[agent]
	if scheme == "" {
		scheme = "http"
	}

	servers := make(map[string]interface{})
	for _, mcp := range appCfg.MCPS {
		url := fmt.Sprintf("%s://%s:%d", scheme, localIP, mcp.Port)
		serverEntry := map[string]interface{}{
			"url": url,
		}
		if agent == "copilot" {
			serverEntry["type"] = scheme
		}
		servers[mcp.Name] = serverEntry
	}

	return map[string]interface{}{
		"servers": servers,
	}, nil
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
  # Output specific agent
  mcpbridgego -o claude                              # Claude config
  mcpbridgego -o copilot                             # Copilot to screen
  mcpbridgego --output generic                       # Generic to screen
  
  # Output agent to file
  mcpbridgego -o claude -f                           # Claude to mcp.json
  mcpbridgego --output copilot --file                # Copilot to mcp.json
  
  # Output to file with custom path
  mcpbridgego -o copilot -f ./agents/copilot-mcp.json
  mcpbridgego -f ./config/mcp-config.json            # Generic to custom path
  
  # Default behavior
  mcpbridgego -o                                     # Same as: -o generic
  mcpbridgego                                        # If no other args, shows config

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
	fmt.Println("  mcpbridgego [options]")
	fmt.Println()
	fmt.Println("Common options:")
	fmt.Println("  -s, --start              Start MCPBridge in background")
	fmt.Println("  -t, --stop               Stop the running MCPBridge")
	fmt.Println("  -r, --run                Run MCPBridge in foreground (no daemon)")
	fmt.Println("  --status                 Check if MCPBridge is running")
	fmt.Println("  -c, --config             Validate the config file")
	fmt.Println("  -h, --help               Show this help message")
	fmt.Println()
	fmt.Println("Output JSON template:")
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

// DisplayAgentCfgInfo displays startup information when MCPBridge starts in background mode
// Shows the dynamic MCP configuration based on the config file
func DisplayAgentCfgInfo() {
	fmt.Println()
	fmt.Printf("%sMCP Configuration for your agents:%s\n", ColorBold, ColorBlue)
	fmt.Println()

	config, err := generateAgentConfig("generic")
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
