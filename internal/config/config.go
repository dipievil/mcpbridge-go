package config

import (
	"fmt"
	"net"
	"os"
	"os/exec"

	"gopkg.in/yaml.v3"
)

// CommandConfig represents a registered command with its path.
type CommandConfig struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

// MCPConfig represents a single MCP (Model Context Protocol) configuration.
type MCPConfig struct {
	Name    string            `yaml:"name"`
	Port    int               `yaml:"port"`
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args"`
	EnvFile string            `yaml:"env_file,omitempty"`
	EnvVars map[string]string `yaml:"env_vars,omitempty"`
	Dir     string            `yaml:"dir,omitempty"`
}

// Config represents the overall configuration with multiple MCPs.
type Config struct {
	Commands []CommandConfig `yaml:"commands"`
	MCPS     []MCPConfig     `yaml:"mcps"`
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v (config.yaml)", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	return &config, nil
}

func GetConfigCommands(config *Config) []CommandConfig {
	if config == nil {
		return nil
	}
	return config.Commands
}

// Validate checks the configuration for required fields and consistency.
func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration is nil")
	}

	if len(cfg.MCPS) == 0 {
		return fmt.Errorf("no MCPs configured in config file")
	}

	if err := validateConfigCommands(cfg); err != nil {
		return err
	}

	for _, mcp := range cfg.MCPS {
		if err := validateMCP(mcp); err != nil {
			return err
		}
	}

	return nil
}

// validateConfigCommands checks all registered commands exist
func validateConfigCommands(cfg *Config) error {
	for _, cmd := range cfg.Commands {
		if err := validateCommand(cmd.Name, cmd.Path); err != nil {
			return err
		}
	}
	return nil
}

// validateCommand checks a single command exists
func validateCommand(commandName string, commandPath string) error {
	if commandName == "" {
		return fmt.Errorf("command name is required in commands registry")
	}
	if commandPath == "" {
		return fmt.Errorf("command path is required for command %s", commandName)
	}
	if _, err := os.Stat(commandPath); os.IsNotExist(err) {
		return fmt.Errorf("command path %s for command %s does not exist", commandPath, commandName)
	}
	return nil
}

// ResolveCommand resolves a command name to its full path.
func ResolveCommand(commandName string) (string, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load config for command resolution: %v", err)
	}

	commands := GetConfigCommands(cfg)

	if len(commandName) > 0 && commandName[0] == '/' {
		for _, cmd := range commands {
			if cmd.Name == commandName {
				return cmd.Path, nil
			}
		}
	}

	if path, err := exec.LookPath(commandName); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("command %s not found in registry or PATH", commandName)
}

// validateMCP checks individual MCP configuration
func validateMCP(mcp MCPConfig) error {
	if mcp.Name == "" {
		return fmt.Errorf("MCP name is required")
	}

	if mcp.Port == 0 {
		return fmt.Errorf("MCP %s port is required", mcp.Name)
	}

	if mcp.Command == "" {
		return fmt.Errorf("MCP %s command is required", mcp.Name)
	}

	// Resolve the command to verify it exists
	if _, err := ResolveCommand(mcp.Command); err != nil {
		return fmt.Errorf("MCP %s: %v", mcp.Name, err)
	}

	if mcp.EnvFile != "" {
		if _, err := os.Stat(mcp.EnvFile); os.IsNotExist(err) {
			return fmt.Errorf("env file %s for MCP %s does not exist. Check yaml file", mcp.EnvFile, mcp.Name)
		}
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", mcp.Port))
	if err != nil {
		return fmt.Errorf("port %d for MCP %s is not available. Check if another process is using it", mcp.Port, mcp.Name)
	}
	ln.Close()

	return nil
}
