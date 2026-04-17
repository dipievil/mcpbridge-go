//go:build e2e
// +build e2e

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestE2E_MCPBridgeWithMockServer(t *testing.T) {
	rootDir, _ := os.Getwd()
	if strings.HasSuffix(rootDir, "/tests") {
		rootDir = filepath.Dir(rootDir)
	}

	mockServerPath := filepath.Join(rootDir, "tests", "mcp_mock_server", "mcp_mock_server")
	bridgeBinaryPath := filepath.Join(rootDir, "bin", "mcpbridgego")
	testDir := filepath.Join(rootDir, "tests", "e2e_test_temp")

	os.RemoveAll(testDir)
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	// Build mock server
	t.Log("Building mock server...")
	cmd := exec.Command("go", "build", "-o", mockServerPath)
	cmd.Dir = filepath.Join(rootDir, "tests", "mcp_mock_server")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build mock server: %v, %s", err, output)
	}
	defer os.Remove(mockServerPath)

	// Build bridge
	t.Log("Building bridge...")
	cmd = exec.Command("go", "build", "-o", bridgeBinaryPath, "./cmd/mcpbridgego")
	cmd.Dir = rootDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build bridge: %v, %s", err, output)
	}

	// Create config
	config := fmt.Sprintf("server:\n  host: \"127.0.0.1\"\nmcps:\n  - name: \"mock\"\n    port: 3000\n    command: \"%s\"\n", mockServerPath)
	configPath := filepath.Join(testDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Start bridge
	t.Log("Starting bridge...")
	bridgeCmd := exec.Command(bridgeBinaryPath)
	bridgeCmd.Dir = testDir
	bridgeCmd.Stdout = os.Stdout
	bridgeCmd.Stderr = os.Stderr

	if err := bridgeCmd.Start(); err != nil {
		t.Fatalf("Failed to start bridge: %v", err)
	}
	defer func() {
		bridgeCmd.Process.Kill()
		bridgeCmd.Wait()
	}()

	// Wait for server to start
	time.Sleep(1 * time.Second)

	// Test 1: Ping
	t.Log("Test 1: ping method...")
	testRequest(t, "http://127.0.0.1:3000/rpc", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "ping",
	})

	// Test 2: Initialize
	t.Log("Test 2: initialize method...")
	result := testRequest(t, "http://127.0.0.1:3000/rpc", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "initialize",
		"params": map[string]string{
			"protocolVersion": "2024-11-05",
		},
	})
	if resultData, ok := result["result"].(map[string]interface{}); !ok || resultData["serverInfo"] == nil {
		t.Fatalf("Initialize response missing serverInfo: %v", result)
	}

	// Test 3: Resources list
	t.Log("Test 3: resources/list method...")
	result = testRequest(t, "http://127.0.0.1:3000/rpc", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "resources/list",
	})
	if _, ok := result["result"]; !ok {
		t.Fatalf("Resources list missing result: %v", result)
	}

	// Test 4: Tools list
	t.Log("Test 4: tools/list method...")
	result = testRequest(t, "http://127.0.0.1:3000/rpc", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "tools/list",
	})
	if _, ok := result["result"]; !ok {
		t.Fatalf("Tools list missing result: %v", result)
	}

	t.Log("✓ All E2E tests passed!")
}

// Helper function to test HTTP requests
func testRequest(t *testing.T, url string, payload interface{}) map[string]interface{} {
	jsonData, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse response: %v, Body: %s", err, string(body))
	}

	if _, ok := result["result"]; !ok {
		t.Fatalf("No result in response: %v", result)
	}

	return result
}
