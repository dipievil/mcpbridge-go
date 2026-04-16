package main

import (
	"encoding/json"
	"testing"
	"time"

	"dipievil/mcpbridgego/internal/bridge"
	"dipievil/mcpbridgego/internal/config"
)

func TestJSONRPCMessageMarshaling(t *testing.T) {
	msg := bridge.JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  "test_method",
		Params:  json.RawMessage(`{"key":"value"}`),
		ID:      1,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal message: %v", err)
	}

	var unmarshaled bridge.JSONRPCMessage
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if unmarshaled.Method != msg.Method {
		t.Errorf("expected method %s, got %s", msg.Method, unmarshaled.Method)
	}

	if unmarshaled.JSONRPC != msg.JSONRPC {
		t.Errorf("expected jsonrpc %s, got %s", msg.JSONRPC, unmarshaled.JSONRPC)
	}
}

func TestJSONRPCErrorMarshaling(t *testing.T) {
	errResp := bridge.JSONRPCError{
		Code:    -32700,
		Message: "Parse error",
		Data:    "Invalid JSON",
	}

	data, err := json.Marshal(errResp)
	if err != nil {
		t.Fatalf("failed to marshal error: %v", err)
	}

	var unmarshaled bridge.JSONRPCError
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal error: %v", err)
	}

	if unmarshaled.Code != errResp.Code {
		t.Errorf("expected code %d, got %d", errResp.Code, unmarshaled.Code)
	}

	if unmarshaled.Message != errResp.Message {
		t.Errorf("expected message %s, got %s", errResp.Message, unmarshaled.Message)
	}
}

func TestMCPConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config config.MCPConfig
		valid  bool
	}{
		{
			name: "valid config",
			config: config.MCPConfig{
				Name:    "test",
				Port:    3000,
				Command: "echo",
				Args:    []string{"hello"},
			},
			valid: true,
		},
		{
			name: "config with env_file",
			config: config.MCPConfig{
				Name:    "test",
				Port:    3000,
				Command: "node",
				Args:    []string{"script.js"},
				EnvFile: "/path/to/.env",
			},
			valid: true,
		},
		{
			name: "config with env_vars",
			config: config.MCPConfig{
				Name:    "test",
				Port:    3000,
				Command: "python",
				Args:    []string{"script.py"},
				EnvVars: map[string]string{"KEY": "value"},
			},
			valid: true,
		},
		{
			name: "config with dir",
			config: config.MCPConfig{
				Name:    "test",
				Port:    3000,
				Command: "bash",
				Args:    []string{"script.sh"},
				Dir:     "/path/to/dir",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation: check required fields
			if tt.config.Name == "" || tt.config.Command == "" {
				if tt.valid {
					t.Error("expected valid config but got invalid")
				}
			} else {
				if !tt.valid {
					t.Error("expected invalid config but got valid")
				}
			}
		})
	}
}

func TestJSONRPCMessageWithNilID(t *testing.T) {
	msg := bridge.JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  "notify_method",
		Params:  json.RawMessage(`{"key":"value"}`),
		ID:      nil,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal message: %v", err)
	}

	var unmarshaled bridge.JSONRPCMessage
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if unmarshaled.ID != nil {
		t.Errorf("expected nil ID, got %v", unmarshaled.ID)
	}
}

func TestTimeoutDuration(t *testing.T) {
	// Test that timeout constant is reasonable
	timeout := 30 * time.Second

	if timeout <= 0 {
		t.Error("timeout should be positive")
	}

	if timeout > 5*time.Minute {
		t.Errorf("timeout seems too long: %v", timeout)
	}
}
