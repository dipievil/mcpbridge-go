package bridge

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

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
	MCPS []MCPConfig `yaml:"mcps"`
}

// JSONRPCMessage represents a JSON-RPC 2.0 message.
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error.
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Bridge manages communication with a single MCP process.
type Bridge struct {
	config        MCPConfig
	mu            sync.Mutex
	stdin         io.WriteCloser
	stdout        io.ReadCloser
	stderr        io.ReadCloser
	cmd           *exec.Cmd
	responseChan  map[interface{}]chan *JSONRPCMessage
	nextMessageID int64
	initialized   bool
}

// NewBridge creates and initializes a new Bridge for a given MCP configuration.
func NewBridge(cfg MCPConfig) (*Bridge, error) {
	b := &Bridge{
		config:       cfg,
		responseChan: make(map[interface{}]chan *JSONRPCMessage),
	}

	log.Printf("Starting MCP %s with command: %s %v", cfg.Name, cfg.Command, cfg.Args)

	envs := make(map[string]string)

	existsEnvFile, err := os.Stat(cfg.EnvFile)
	if err != nil || existsEnvFile.IsDir() {
		log.Printf("Warning: env file %s for MCP %s does not exist or is a directory. Skipping env file loading.", cfg.EnvFile, cfg.Name)
	} else {
		envs, _ = godotenv.Read(cfg.EnvFile)
		maps.Copy(envs, cfg.EnvVars)
	}

	maps.Copy(envs, cfg.EnvVars)

	log.Printf("Environment variables for MCP %s: %v", cfg.Name, envs)

	cmd := exec.Command(cfg.Command, cfg.Args...)
	if cfg.Dir != "" {
		cmd.Dir = cfg.Dir
	}
	cmd.Env = os.Environ()
	for k, v := range envs {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start MCP %s: %v", cfg.Name, err)
	}

	b.cmd = cmd
	b.stdin = stdin
	b.stdout = stdout
	b.stderr = stderr

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("[%s stderr] %s", cfg.Name, scanner.Text())
		}
	}()

	go b.readMessages()

	log.Printf("MCP %s started on pid %d", cfg.Name, cmd.Process.Pid)
	b.initialized = true
	return b, nil
}

// readMessages reads JSON-RPC messages from MCP's stdout in background and dispatches them to waiting clients.
func (b *Bridge) readMessages() {
	scanner := bufio.NewScanner(b.stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var msg JSONRPCMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			log.Printf("[%s] Error parsing message: %v", b.config.Name, err)
			continue
		}

		b.mu.Lock()
		if ch, exists := b.responseChan[msg.ID]; exists && msg.ID != nil {
			ch <- &msg
			delete(b.responseChan, msg.ID)
		} else {
			for id, respCh := range b.responseChan {
				select {
				case respCh <- &msg:
					delete(b.responseChan, id)
				default:
				}
			}
		}
		b.mu.Unlock()
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[%s] Error reading stdout: %v", b.config.Name, err)
	}
}

// SendMessage sends a JSON-RPC message to the MCP and waits for response.
func (b *Bridge) SendMessage(msg *JSONRPCMessage, timeout time.Duration) (*JSONRPCMessage, error) {
	if !b.initialized {
		return nil, fmt.Errorf("bridge not initialized")
	}

	if msg.ID == nil && msg.Method != "" {
		b.mu.Lock()
		b.nextMessageID++
		msg.ID = b.nextMessageID
		b.mu.Unlock()
	}

	responseCh := make(chan *JSONRPCMessage, 1)
	if msg.ID != nil {
		b.mu.Lock()
		b.responseChan[msg.ID] = responseCh
		b.mu.Unlock()
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %v", err)
	}

	b.mu.Lock()
	_, err = b.stdin.Write(append(data, '\n'))
	b.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to write to stdin: %v", err)
	}

	ctx := time.After(timeout)
	select {
	case resp := <-responseCh:
		return resp, nil
	case <-ctx:
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

// Close closes the bridge and MCP process.
func (b *Bridge) Close() error {
	if b.cmd != nil && b.cmd.Process != nil {
		b.cmd.Process.Kill()
	}
	if b.stdin != nil {
		b.stdin.Close()
	}
	return nil
}

// HandleRPC handles JSON-RPC method calls.
func (b *Bridge) HandleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodOptions {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var reqMsg JSONRPCMessage
	if err := json.NewDecoder(r.Body).Decode(&reqMsg); err != nil {
		errResp := JSONRPCMessage{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    -32700,
				Message: "Parse error",
			},
			ID: nil,
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	if reqMsg.JSONRPC == "" {
		reqMsg.JSONRPC = "2.0"
	}

	respMsg, err := b.SendMessage(&reqMsg, 30*time.Second)
	if err != nil {
		errResp := JSONRPCMessage{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    -32603,
				Message: fmt.Sprintf("Internal error: %v", err),
			},
			ID: reqMsg.ID,
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(respMsg)
}

// HandleSSE handles Server-Sent Events streaming.
func (b *Bridge) HandleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	w.WriteHeader(http.StatusOK)
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}
	flusher.Flush()

	responseCh := make(chan *JSONRPCMessage, 100)
	clientID := time.Now().UnixNano()

	b.mu.Lock()
	b.responseChan[clientID] = responseCh
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		delete(b.responseChan, clientID)
		close(responseCh)
		b.mu.Unlock()
	}()

	for {
		select {
		case msg, ok := <-responseCh:
			if !ok {
				return
			}
			data, _ := json.Marshal(msg)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// HandleHealth returns server health status.
func (b *Bridge) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	pid := 0
	if b.cmd != nil && b.cmd.Process != nil {
		pid = b.cmd.Process.Pid
	}

	health := map[string]interface{}{
		"status": "ok",
		"mcp":    b.config.Name,
		"pid":    pid,
		"port":   b.config.Port,
	}
	json.NewEncoder(w).Encode(health)
}
