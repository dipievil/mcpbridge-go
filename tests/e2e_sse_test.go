//go:build e2e
// +build e2e

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestE2E_MCPBridgeSSEStreaming tests the bridge's Server-Sent Events streaming capability
func TestE2E_MCPBridgeSSEStreaming(t *testing.T) {
	rootDir, _ := os.Getwd()
	if strings.HasSuffix(rootDir, "/tests") {
		rootDir = filepath.Dir(rootDir)
	}

	mockServerPath := filepath.Join(rootDir, "tests", "mcp_mock_server", "mcp_mock_server_sse")
	bridgeBinaryPath := filepath.Join(rootDir, "bin", "mcpbridgego_sse")
	testDir := filepath.Join(rootDir, "tests", "e2e_sse_test_temp")

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
	defer os.Remove(bridgeBinaryPath)

	// Create config
	configPath := filepath.Join(testDir, "config.yaml")
	configContent := fmt.Sprintf(`server:
  host: "127.0.0.1"

mcps:
  - name: "test-mcp"
    port: 3001
    command: "%s"
`, mockServerPath)

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Start bridge
	t.Log("Starting bridge...")
	bridgeCmd := exec.Command(bridgeBinaryPath, configPath)
	bridgeCmd.Dir = testDir
	if err := bridgeCmd.Start(); err != nil {
		t.Fatalf("Failed to start bridge: %v", err)
	}
	defer bridgeCmd.Process.Kill()

	// Wait for server to start
	time.Sleep(1 * time.Second)

	t.Log("✓ Build complete: bridge and mock server")

	// Test SSE streaming
	t.Run("SSE_Ping", func(t *testing.T) {
		testSSERequest(t, "http://127.0.0.1:3001/sse", map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "ping",
			"id":      1,
		})
		t.Log("  ✓ SSE ping method works")
	})

	t.Run("SSE_Initialize", func(t *testing.T) {
		testSSERequest(t, "http://127.0.0.1:3001/sse", map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "initialize",
			"id":      2,
		})
		t.Log("  ✓ SSE initialize method works")
	})

	t.Run("SSE_ResourcesList", func(t *testing.T) {
		testSSERequest(t, "http://127.0.0.1:3001/sse", map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "resources/list",
			"id":      3,
		})
		t.Log("  ✓ SSE resources/list method works")
	})

	t.Run("SSE_ToolsList", func(t *testing.T) {
		testSSERequest(t, "http://127.0.0.1:3001/sse", map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "tools/list",
			"id":      4,
		})
		t.Log("  ✓ SSE tools/list method works")
	})

	t.Log("✓ All SSE E2E tests passed!")
}

// testSSERequest sends a request via SSE and reads the response using raw TCP
func testSSERequest(t *testing.T, url string, payload interface{}) map[string]interface{} {
	// Parse URL to get host:port
	urlParts := strings.Split(strings.TrimPrefix(url, "http://"), "/")
	hostPort := urlParts[0]
	path := "/" + strings.Join(urlParts[1:], "/")

	// Connect to server directly
	conn, err := net.Dial("tcp", hostPort)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Set timeout on connection
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))

	// Build HTTP request
	data, _ := json.Marshal(payload)
	request := fmt.Sprintf("POST %s HTTP/1.1\r\n", path)
	request += fmt.Sprintf("Host: %s\r\n", hostPort)
	request += "Content-Type: application/json\r\n"
	request += fmt.Sprintf("Content-Length: %d\r\n", len(data))
	request += "Connection: close\r\n"
	request += "\r\n"

	// Send request
	if _, err := conn.Write([]byte(request)); err != nil {
		t.Fatalf("Failed to write headers: %v", err)
	}

	if _, err := conn.Write(data); err != nil {
		t.Fatalf("Failed to write body: %v", err)
	}

	// Read response
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil && err.Error() != "EOF" && !strings.Contains(err.Error(), "use of closed network connection") {
		t.Fatalf("Failed to read response: %v", err)
	}

	if n == 0 {
		t.Fatalf("No data read from response")
	}

	response := string(buf[:n])

	// Parse HTTP response to extract body
	parts := strings.SplitN(response, "\r\n\r\n", 2)
	if len(parts) < 2 {
		t.Fatalf("Invalid HTTP response format: %s", response)
	}

	body := parts[1]

	// Parse SSE response
	lines := strings.Split(body, "\n")
	var result map[string]interface{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data: ") {
			jsonStr := strings.TrimPrefix(line, "data: ")
			if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
				t.Fatalf("Failed to unmarshal SSE response: %v, body was: %s", err, body)
				return nil
			}

			// Verify it's a valid JSON-RPC response
			if result["jsonrpc"] != "2.0" {
				t.Fatalf("Invalid JSON-RPC response: %v", result)
			}

			return result
		}
	}

	t.Fatalf("No SSE data found in response. Got: %s", body)
	return nil
}
