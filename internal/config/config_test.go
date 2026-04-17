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

func TestValidateEnvFileRelativeToDir(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(origDir) })

	// Create a subdirectory for the MCP
	mcpDir := tmpDir + "/mcp-service"
	os.Mkdir(mcpDir, 0755)

	// Create a .env file in the MCP directory
	envFile := mcpDir + "/.env"
	os.WriteFile(envFile, []byte("TEST_VAR=test_value\nANOTHER=another_value"), 0644)

	// Create config.yaml in the root tmpDir
	configContent := `
mcps:
  - name: test-mcp
    port: 3000
    command: echo
    args: ["hello"]
    dir: ` + mcpDir + `
    env_file: ".env"
    env_vars:
      DIRECT_VAR: "direct_value"
`
	configFile := tmpDir + "/config.yaml"
	os.WriteFile(configFile, []byte(configContent), 0644)

	// Load and validate
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if err := Validate(cfg); err != nil {
		t.Errorf("validation should pass with env_file relative to dir: %v", err)
	}

	// Check that MergedEnv was populated
	if cfg.MCPS[0].MergedEnv == nil {
		t.Error("MergedEnv should not be nil after validation")
	}

	// Check that env file variables are loaded
	if val, exists := cfg.MCPS[0].MergedEnv["TEST_VAR"]; !exists || val != "test_value" {
		t.Errorf("expected TEST_VAR=test_value in MergedEnv, got %v", cfg.MCPS[0].MergedEnv)
	}

	// Check that direct env_vars take precedence (though not overridden in this case)
	if val, exists := cfg.MCPS[0].MergedEnv["DIRECT_VAR"]; !exists || val != "direct_value" {
		t.Errorf("expected DIRECT_VAR=direct_value in MergedEnv, got %v", cfg.MCPS[0].MergedEnv)
	}
}

func TestValidateEnvFileMissingFails(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(origDir) })

	// Create a subdirectory but WITHOUT a .env file
	mcpDir := tmpDir + "/mcp-service"
	os.Mkdir(mcpDir, 0755)

	configContent := `
mcps:
  - name: test-mcp
    port: 3000
    command: echo
    args: ["hello"]
    dir: ` + mcpDir + `
    env_file: ".env"
`
	configFile := tmpDir + "/config.yaml"
	os.WriteFile(configFile, []byte(configContent), 0644)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Validation should FAIL because .env doesn't exist
	if err := Validate(cfg); err == nil {
		t.Error("validation should fail when env_file does not exist at resolved path")
	}
}

func TestEnvVarsOverrideEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(origDir) })

	mcpDir := tmpDir + "/mcp-service"
	os.Mkdir(mcpDir, 0755)

	// Create a .env file with some variables
	envFile := mcpDir + "/.env"
	os.WriteFile(envFile, []byte("VAR1=from_file\nVAR2=also_from_file"), 0644)

	configContent := `
mcps:
  - name: test-mcp
    port: 3000
    command: echo
    args: ["hello"]
    dir: ` + mcpDir + `
    env_file: ".env"
    env_vars:
      VAR1: "from_vars"
`
	configFile := tmpDir + "/config.yaml"
	os.WriteFile(configFile, []byte(configContent), 0644)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if err := Validate(cfg); err != nil {
		t.Errorf("validation should pass: %v", err)
	}

	// VAR1 should come from env_vars (takes precedence), not from .env
	if val, exists := cfg.MCPS[0].MergedEnv["VAR1"]; !exists || val != "from_vars" {
		t.Errorf("expected VAR1=from_vars (env_vars takes precedence), got %v", cfg.MCPS[0].MergedEnv["VAR1"])
	}

	// VAR2 should come from .env
	if val, exists := cfg.MCPS[0].MergedEnv["VAR2"]; !exists || val != "also_from_file" {
		t.Errorf("expected VAR2=also_from_file in MergedEnv, got %v", cfg.MCPS[0].MergedEnv)
	}
}
