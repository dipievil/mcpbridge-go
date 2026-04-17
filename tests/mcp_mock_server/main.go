package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
)

// JSONRPCMessage represents a JSON-RPC 2.0 message
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

var messageCounter int64

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg JSONRPCMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing message: %v\n", err)
			continue
		}

		// Handle initialize method
		if msg.Method == "initialize" {
			response := JSONRPCMessage{
				JSONRPC: "2.0",
				ID:      msg.ID,
				Result:  json.RawMessage(`{"protocolVersion":"2024-11-05","capabilities":{},"serverInfo":{"name":"mock-server","version":"1.0.0"}}`),
			}
			if data, err := json.Marshal(response); err == nil {
				fmt.Println(string(data))
			}
			continue
		}

		// Handle ping method
		if msg.Method == "ping" {
			response := JSONRPCMessage{
				JSONRPC: "2.0",
				ID:      msg.ID,
				Result:  json.RawMessage(`{}`),
			}
			if data, err := json.Marshal(response); err == nil {
				fmt.Println(string(data))
			}
			continue
		}

		// Handle resources/list method
		if msg.Method == "resources/list" {
			response := JSONRPCMessage{
				JSONRPC: "2.0",
				ID:      msg.ID,
				Result:  json.RawMessage(`{"resources":[{"uri":"file:///test","name":"test-resource"}]}`),
			}
			if data, err := json.Marshal(response); err == nil {
				fmt.Println(string(data))
			}
			continue
		}

		// Handle tools/list method
		if msg.Method == "tools/list" {
			response := JSONRPCMessage{
				JSONRPC: "2.0",
				ID:      msg.ID,
				Result:  json.RawMessage(`{"tools":[{"name":"test-tool","description":"A test tool","inputSchema":{"type":"object"}}]}`),
			}
			if data, err := json.Marshal(response); err == nil {
				fmt.Println(string(data))
			}
			continue
		}

		// Echo back any other method
		msgID := atomic.AddInt64(&messageCounter, 1)
		response := JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      msgID,
			Result:  json.RawMessage(`{"status":"ok"}`),
		}
		if data, err := json.Marshal(response); err == nil {
			fmt.Println(string(data))
		}
	}
}
