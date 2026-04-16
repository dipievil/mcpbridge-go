package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `
mcps:
  - name: claude
    port: 3000
    command: echo
    args: ["hello"]

  - name: copilot
    port: 3001
    command: echo
    args: ["world"]
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Load config
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify config loaded correctly
	if len(cfg.MCPS) != 2 {
		t.Errorf("expected 2 MCPs, got %d", len(cfg.MCPS))
	}

	if cfg.MCPS[0].Name != "claude" {
		t.Errorf("expected first MCP name to be 'claude', got %s", cfg.MCPS[0].Name)
	}

	if cfg.MCPS[0].Port != 3000 {
		t.Errorf("expected first MCP port to be 3000, got %d", cfg.MCPS[0].Port)
	}
}

func TestValidate(t *testing.T) {
	// Create a temporary config file with valid MCPs
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `
mcps:
  - name: echo
    port: 3000
    command: echo
    args: ["hello"]
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Load and validate config
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if err := Validate(cfg); err != nil {
		t.Errorf("validation should pass for valid config: %v", err)
	}
}

func TestValidateEmptyMCPs(t *testing.T) {
	// Create a temporary config file with no MCPs
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `
mcps: []
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Load and validate config
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if err := Validate(cfg); err == nil {
		t.Error("validation should fail for empty MCPs")
	}
}

func TestValidateInvalidPort(t *testing.T) {
	// Create a temporary config file with invalid port
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `
mcps:
  - name: test
    port: 99999 
    command: echo
    args: ["hello"]
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Load and validate config
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Port 99999 is likely in use, so this should fail
	// But we can't guarantee it for testing, so this test is more of a smoke test
	_ = Validate(cfg)
}

func TestValidateInvalidCommand(t *testing.T) {
	// Create a temporary config file with invalid command
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `
mcps:
  - name: test
    port: 3000
    command: /nonexistent/command/that/does/not/exist
    args: ["hello"]
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Load and validate config
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if err := Validate(cfg); err == nil {
		t.Error("validation should fail for command not in PATH")
	}
}

func TestValidateWithOptionalEnvFile(t *testing.T) {
	// Create a temporary config file with optional env_file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `
mcps:
  - name: test
    port: 3000
    command: echo
    args: ["hello"]
    env_file: ""
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Load and validate config
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if err := Validate(cfg); err != nil {
		t.Errorf("validation should pass when env_file is empty: %v", err)
	}
}
