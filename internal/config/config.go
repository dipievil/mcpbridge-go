package config

import (
	"fmt"
	"net"
	"os"
	"os/exec"

	"dipievil/mcpbridgego/internal/bridge"

	"gopkg.in/yaml.v3"
)

// LoadConfig reads and parses a YAML config file
func LoadConfig(configFile string) (*bridge.Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v (%s)", err, configFile)
	}

	var config bridge.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	return &config, nil
}

// Validate checks if the configuration is valid
func Validate(cfg *bridge.Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration is nil")
	}

	if len(cfg.MCPS) == 0 {
		return fmt.Errorf("no MCPs configured in config file")
	}

	// Validate each MCP
	for _, mcp := range cfg.MCPS {
		if err := validateMCP(mcp); err != nil {
			return err
		}
	}

	return nil
}

// validateMCP checks individual MCP configuration
func validateMCP(mcp bridge.MCPConfig) error {
	if mcp.Name == "" {
		return fmt.Errorf("MCP name is required")
	}

	if mcp.Port == 0 {
		return fmt.Errorf("MCP %s port is required", mcp.Name)
	}

	if mcp.Command == "" {
		return fmt.Errorf("MCP %s command is required", mcp.Name)
	}

	// Only check env_file if it's specified
	if mcp.EnvFile != "" {
		if _, err := os.Stat(mcp.EnvFile); os.IsNotExist(err) {
			return fmt.Errorf("env file %s for MCP %s does not exist. Check yaml file", mcp.EnvFile, mcp.Name)
		}
	}

	// Check if port is available
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", mcp.Port))
	if err != nil {
		return fmt.Errorf("port %d for MCP %s is not available. Check if another process is using it", mcp.Port, mcp.Name)
	}
	ln.Close()

	// Check if command exists in PATH
	if _, err := exec.LookPath(mcp.Command); err != nil {
		return fmt.Errorf("command %s for MCP %s not found in PATH. Check yaml file", mcp.Command, mcp.Name)
	}

	return nil
}
