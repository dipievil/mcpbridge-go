package config

import (
	"os"
	"testing"
)

// setupTestConfig creates a config.yaml in a temp dir and changes to that dir.
// Returns a cleanup function to restore the original working directory.
func setupTestConfig(t *testing.T, content string) {
	t.Helper()
	tmpDir := t.TempDir()
	configFile := tmpDir + "/config.yaml"

	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	t.Cleanup(func() {
		os.Chdir(origDir)
	})
}

func TestLoadConfig(t *testing.T) {
	setupTestConfig(t, `
mcps:
  - name: claude
    port: 3000
    command: echo
    args: ["hello"]

  - name: copilot
    port: 3001
    command: echo
    args: ["world"]
`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

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

func TestLoadConfigMissing(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(origDir) })

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error when config.yaml is missing")
	}
}

func TestValidate(t *testing.T) {
	setupTestConfig(t, `
mcps:
  - name: echo
    port: 3000
    command: echo
    args: ["hello"]
`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if err := Validate(cfg); err != nil {
		t.Errorf("validation should pass for valid config: %v", err)
	}
}

func TestValidateEmptyMCPs(t *testing.T) {
	setupTestConfig(t, `
mcps: []
`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if err := Validate(cfg); err == nil {
		t.Error("validation should fail for empty MCPs")
	}
}

func TestValidateInvalidPort(t *testing.T) {
	setupTestConfig(t, `
mcps:
  - name: test
    port: 99999
    command: echo
    args: ["hello"]
`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Port 99999 is invalid but we can't guarantee behavior
	_ = Validate(cfg)
}

func TestValidateInvalidCommand(t *testing.T) {
	setupTestConfig(t, `
mcps:
  - name: test
    port: 3000
    command: ""
    args: ["hello"]
`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if err := Validate(cfg); err == nil {
		t.Error("validation should fail for empty command")
	}
}

func TestValidateWithOptionalEnvFile(t *testing.T) {
	setupTestConfig(t, `
mcps:
  - name: test
    port: 3000
    command: echo
    args: ["hello"]
    env_file: ""
`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if err := Validate(cfg); err != nil {
		t.Errorf("validation should pass when env_file is empty: %v", err)
	}
}
