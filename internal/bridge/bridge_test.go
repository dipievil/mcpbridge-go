package bridge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dipievil/mcpbridgego/internal/config"
)

// TestJSONRPCMessage tests the JSONRPCMessage structure
func TestJSONRPCMessage(t *testing.T) {
	msg := JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  "initialize",
		ID:      1,
	}

	if msg.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC 2.0, got %s", msg.JSONRPC)
	}
	if msg.Method != "initialize" {
		t.Errorf("expected method initialize, got %s", msg.Method)
	}
	if msg.ID != 1 {
		t.Errorf("expected ID 1, got %v", msg.ID)
	}
}

// TestJSONRPCError tests the JSONRPCError structure
func TestJSONRPCError(t *testing.T) {
	errMsg := JSONRPCError{
		Code:    -32700,
		Message: "Parse error",
	}

	if errMsg.Code != -32700 {
		t.Errorf("expected code -32700, got %d", errMsg.Code)
	}
	if errMsg.Message != "Parse error" {
		t.Errorf("expected message 'Parse error', got %s", errMsg.Message)
	}
}

// TestMCPConfig tests the MCPConfig structure
func TestMCPConfig(t *testing.T) {
	cfg := config.MCPConfig{
		Name:    "test-mcp",
		Port:    3000,
		Command: "echo",
	}

	if cfg.Name != "test-mcp" {
		t.Errorf("expected name 'test-mcp', got %s", cfg.Name)
	}
	if cfg.Port != 3000 {
		t.Errorf("expected port 3000, got %d", cfg.Port)
	}
	if cfg.Command != "echo" {
		t.Errorf("expected command 'echo', got %s", cfg.Command)
	}
}

// TestConfig tests the Config structure
func TestConfig(t *testing.T) {
	cfg := config.Config{
		MCPS: []config.MCPConfig{
			{
				Name:    "mcp-1",
				Port:    3000,
				Command: "echo",
			},
			{
				Name:    "mcp-2",
				Port:    3001,
				Command: "echo",
			},
		},
	}

	if len(cfg.MCPS) != 2 {
		t.Errorf("expected 2 MCPs, got %d", len(cfg.MCPS))
	}
	if cfg.MCPS[0].Name != "mcp-1" {
		t.Errorf("expected first MCP name 'mcp-1', got %s", cfg.MCPS[0].Name)
	}
	if cfg.MCPS[1].Port != 3001 {
		t.Errorf("expected second MCP port 3001, got %d", cfg.MCPS[1].Port)
	}
}

// TestHandleRPC_MethodNotAllowed tests HandleRPC with unsupported method
func TestHandleRPC_MethodNotAllowed(t *testing.T) {
	cfg := config.MCPConfig{
		Name:    "test-mcp",
		Port:    3000,
		Command: "echo",
	}
	bridge := &Bridge{
		config:        cfg,
		responseChan:  make(map[string]chan *JSONRPCMessage),
		initialized:   true,
		nextMessageID: 0,
	}

	req := httptest.NewRequest("DELETE", "/rpc", nil)
	w := httptest.NewRecorder()

	bridge.HandleRPC(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// TestHandleRPC_CORSHeaders tests that HandleRPC sets CORS headers
func TestHandleRPC_CORSHeaders(t *testing.T) {
	cfg := config.MCPConfig{
		Name:    "test-mcp",
		Port:    3000,
		Command: "echo",
	}
	bridge := &Bridge{
		config:        cfg,
		responseChan:  make(map[string]chan *JSONRPCMessage),
		initialized:   true,
		nextMessageID: 0,
	}

	req := httptest.NewRequest("OPTIONS", "/rpc", nil)
	w := httptest.NewRecorder()

	bridge.HandleRPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected CORS header '*', got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %s", w.Header().Get("Content-Type"))
	}
}

// TestHandleRPC_InvalidJSON tests HandleRPC with invalid JSON
func TestHandleRPC_InvalidJSON(t *testing.T) {
	cfg := config.MCPConfig{
		Name:    "test-mcp",
		Port:    3000,
		Command: "echo",
	}
	bridge := &Bridge{
		config:        cfg,
		responseChan:  make(map[string]chan *JSONRPCMessage),
		initialized:   true,
		nextMessageID: 0,
	}

	invalidJSON := `{invalid json}`
	req := httptest.NewRequest("POST", "/rpc", strings.NewReader(invalidJSON))
	w := httptest.NewRecorder()

	bridge.HandleRPC(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Verify error response structure
	var resp JSONRPCMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Errorf("expected error in response, got nil")
	}
	if resp.Error.Code != -32700 {
		t.Errorf("expected error code -32700, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "Parse error" {
		t.Errorf("expected error message 'Parse error', got %s", resp.Error.Message)
	}
}

// TestHandleHealth tests the HandleHealth handler
func TestHandleHealth(t *testing.T) {
	cfg := config.MCPConfig{
		Name:    "test-mcp",
		Port:    3000,
		Command: "echo",
	}
	b := &Bridge{
		config:        cfg,
		responseChan:  make(map[string]chan *JSONRPCMessage),
		initialized:   true,
		nextMessageID: 0,
	}

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	b.HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify response structure
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", resp["status"])
	}
	if resp["mcp"] != "test-mcp" {
		t.Errorf("expected mcp 'test-mcp', got %v", resp["mcp"])
	}
	if resp["port"] != float64(3000) {
		t.Errorf("expected port 3000, got %v", resp["port"])
	}
}

// TestBridgeInitialized tests that bridge is properly initialized
func TestBridgeInitialized(t *testing.T) {
	cfg := config.MCPConfig{
		Name:    "test-mcp",
		Port:    3000,
		Command: "echo",
	}
	bridge := &Bridge{
		config:        cfg,
		responseChan:  make(map[string]chan *JSONRPCMessage),
		nextMessageID: 0,
		initialized:   false,
	}

	msg := &JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  "test",
		ID:      1,
	}

	// Should fail because bridge is not initialized
	_, err := bridge.SendMessage(msg, 1*time.Second)
	if err == nil {
		t.Errorf("expected error for uninitialized bridge, got nil")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("expected error containing 'not initialized', got %v", err)
	}
}

// TestClose tests the Close method
func TestClose(t *testing.T) {
	cfg := config.MCPConfig{
		Name:    "test-mcp",
		Port:    3000,
		Command: "echo",
	}
	b := &Bridge{
		config:        cfg,
		responseChan:  make(map[string]chan *JSONRPCMessage),
		nextMessageID: 0,
		initialized:   true,
	}

	err := b.Close()
	if err != nil {
		t.Errorf("expected no error on close, got %v", err)
	}
}
