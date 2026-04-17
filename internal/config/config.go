package config

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/joho/godotenv"
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
	// MergedEnv contains all environment variables ready to use, loaded from env_file + env_vars during validation
	MergedEnv map[string]string `yaml:"-"`
}

// Config represents the overall configuration with multiple MCPs.
type Config struct {
	Commands []CommandConfig `yaml:"commands"`
	MCPS     []MCPConfig     `yaml:"mcps"`
}

func LoadConfig() (*Config, error) {
	const configFile = "config.yaml"

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s. Create a config.yaml in the current directory", configFile)
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v (%s)", err, configFile)
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

	for i := range cfg.MCPS {
		if err := validateMCP(&cfg.MCPS[i]); err != nil {
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

	for _, cmd := range commands {
		if cmd.Name == commandName {
			return cmd.Path, nil
		}
	}

	if len(commandName) > 0 && commandName[0] == '/' {
		if _, err := os.Stat(commandName); err == nil {
			return commandName, nil
		}
	}

	if path, err := exec.LookPath(commandName); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("command %s not found in registry or PATH", commandName)
}

// validateMCP checks individual MCP configuration and loads merged environment variables
func validateMCP(mcp *MCPConfig) error {
	if mcp.Name == "" {
		return fmt.Errorf("MCP name is required")
	}

	if mcp.Port == 0 {
		return fmt.Errorf("MCP %s port is required", mcp.Name)
	}

	if mcp.Command == "" {
		return fmt.Errorf("MCP %s command is required", mcp.Name)
	}

	if err := loadMergedEnv(mcp); err != nil {
		return err
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", mcp.Port))
	if err != nil {
		return fmt.Errorf("port %d for MCP %s is not available. Check if another process is using it", mcp.Port, mcp.Name)
	}
	ln.Close()

	return nil
}

// loadMergedEnv loads environment variables from env_file (resolved relative to Dir) and merges with env_vars
func loadMergedEnv(mcp *MCPConfig) error {
	mcp.MergedEnv = make(map[string]string)
	for k, v := range mcp.EnvVars {
		mcp.MergedEnv[k] = v
	}

	if mcp.EnvFile != "" {
		envFilePath := mcp.EnvFile

		if !filepath.IsAbs(envFilePath) && mcp.Dir != "" {
			envFilePath = filepath.Join(mcp.Dir, envFilePath)
		}

		if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
			return fmt.Errorf("env file %s for MCP %s does not exist. Check yaml file", envFilePath, mcp.Name)
		} else if err != nil {
			return fmt.Errorf("error accessing env file %s for MCP %s: %v", envFilePath, mcp.Name, err)
		}

		envs, err := godotenv.Read(envFilePath)
		if err != nil {
			return fmt.Errorf("error reading env file %s for MCP %s: %v", envFilePath, mcp.Name, err)
		}

		for k, v := range envs {
			if _, exists := mcp.EnvVars[k]; !exists {
				mcp.MergedEnv[k] = v
			}
		}
	}

	return nil
}
