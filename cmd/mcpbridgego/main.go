package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type MCPConfig struct {
	Name    string   `yaml:"name"`
	Port    int      `yaml:"port"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
	EnvFile string   `yaml:"env_file"`
	Dir     string   `yaml:"dir"`
}

type Config struct {
	MCPS []MCPConfig `yaml:"mcps"`
}

// JSONRPCMessage represents a JSON-RPC 2.0 message
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Bridge maintains a single persistent connection to an MCP server
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

// NewBridge creates a new bridge and starts the MCP process
func NewBridge(cfg MCPConfig) (*Bridge, error) {
	b := &Bridge{
		config:       cfg,
		responseChan: make(map[interface{}]chan *JSONRPCMessage),
	}

	envs, _ := godotenv.Read(cfg.EnvFile)

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

	// Log stderr in background
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("[%s stderr] %s", cfg.Name, scanner.Text())
		}
	}()

	// Read messages from MCP in background
	go b.readMessages()

	log.Printf("MCP %s started on pid %d", cfg.Name, cmd.Process.Pid)
	b.initialized = true
	return b, nil
}

// readMessages reads JSON-RPC messages from MCP's stdout
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
			// Broadcast to all waiting clients if no specific ID match
			// (for server push notifications)
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

// SendMessage sends a JSON-RPC message to the MCP and waits for response
func (b *Bridge) SendMessage(msg *JSONRPCMessage, timeout time.Duration) (*JSONRPCMessage, error) {
	if !b.initialized {
		return nil, fmt.Errorf("bridge not initialized")
	}

	// Set default ID if not provided
	if msg.ID == nil && msg.Method != "" {
		b.mu.Lock()
		b.nextMessageID++
		msg.ID = b.nextMessageID
		b.mu.Unlock()
	}

	// Create response channel
	responseCh := make(chan *JSONRPCMessage, 1)
	if msg.ID != nil {
		b.mu.Lock()
		b.responseChan[msg.ID] = responseCh
		b.mu.Unlock()
	}

	// Send message
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

	// Wait for response
	ctx := time.After(timeout)
	select {
	case resp := <-responseCh:
		return resp, nil
	case <-ctx:
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

// Close closes the bridge and MCP process
func (b *Bridge) Close() error {
	if b.cmd != nil && b.cmd.Process != nil {
		b.cmd.Process.Kill()
	}
	if b.stdin != nil {
		b.stdin.Close()
	}
	return nil
}

// handleRPC handles JSON-RPC method calls
func (b *Bridge) handleRPC(w http.ResponseWriter, r *http.Request) {
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

	// Set default JSONRPC version if not provided
	if reqMsg.JSONRPC == "" {
		reqMsg.JSONRPC = "2.0"
	}

	// Forward message to MCP and wait for response
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

// handleSSE handles Server-Sent Events streaming
func (b *Bridge) handleSSE(w http.ResponseWriter, r *http.Request) {
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

	// Create a client for receiving messages
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

	// Send messages to client as they arrive
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

// handleHealth returns server health status
func (b *Bridge) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	health := map[string]interface{}{
		"status": "ok",
		"mcp":    b.config.Name,
		"pid":    b.cmd.Process.Pid,
		"port":   b.config.Port,
	}
	json.NewEncoder(w).Encode(health)
}

// getPIDFile returns the path to the PID file
func getPIDFile() string {
	pidFile := "/var/run/mcpbridgego.pid"
	if err := os.WriteFile(pidFile, []byte("test"), 0644); err == nil {
		os.Remove(pidFile)
		return pidFile
	}
	return filepath.Join(os.TempDir(), "mcpbridgego.pid")
}

// isProcessRunning checks if a process with the given PID is still running
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	defer process.Release()

	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// savePID writes the current process ID to a file
func savePID() error {
	pidFile := getPIDFile()
	pid := os.Getpid()
	return os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)
}

// readPID reads the PID from the PID file
func readPID() (int, error) {
	pidFile := getPIDFile()
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

// removePIDFile removes the PID file
func removePIDFile() error {
	pidFile := getPIDFile()
	return os.Remove(pidFile)
}

// startDaemon starts the app in background
func startDaemon(configFile string) error {
	pidFile := getPIDFile()

	if pid, err := readPID(); err == nil {
		if isProcessRunning(pid) {
			return fmt.Errorf("MCPBridge is already running (PID: %d)", pid)
		}
		removePIDFile()
	}

	cmd := exec.Command(os.Args[0], configFile)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCPBridge: %v", err)
	}

	pidData := []byte(strconv.Itoa(cmd.Process.Pid))
	if err := os.WriteFile(pidFile, pidData, 0644); err != nil {
		log.Printf("Warning: could not write PID file: %v", err)
	}

	fmt.Printf("MCPBridge started in background (PID: %d)\n", cmd.Process.Pid)
	os.Exit(0)
	return nil
}

// stopDaemon stops the running background process
func stopDaemon() error {
	pid, err := readPID()
	if err != nil {
		return fmt.Errorf("MCPBridge is not running (no PID file found)")
	}

	if !isProcessRunning(pid) {
		removePIDFile()
		return fmt.Errorf("MCPBridge is not running (PID %d not found)", pid)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %v", err)
	}
	defer process.Release()

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to stop MCPBridge: %v", err)
	}

	removePIDFile()
	fmt.Printf("MCPBridge stopped (PID: %d)\n", pid)
	return nil
}

// runForeground runs the app in foreground mode
func runForeground(configFile string) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	var tempConfig Config
	yaml.Unmarshal(data, &tempConfig)
	for _, mcp := range tempConfig.MCPS {
		if _, err := os.Stat(mcp.EnvFile); os.IsNotExist(err) {
			return fmt.Errorf("env file %s for MCP %s does not exist. Check yaml file", mcp.EnvFile, mcp.Name)
		}
	}

	for _, mcp := range tempConfig.MCPS {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", mcp.Port))
		if err != nil {
			return fmt.Errorf("port %d for MCP %s is not available. Check if another process is using it", mcp.Port, mcp.Name)
		}
		ln.Close()
	}

	for _, mcp := range tempConfig.MCPS {
		if _, err := exec.LookPath(mcp.Command); err != nil {
			return fmt.Errorf("command %s for MCP %s not found in PATH. Check yaml file", mcp.Command, mcp.Name)
		}
	}

	var config Config
	yaml.Unmarshal(data, &config)

	if err := savePID(); err != nil {
		log.Printf("Warning: could not write PID file: %v", err)
	}

	// Start all MCP bridges
	var bridges []*Bridge
	for _, mcp := range config.MCPS {
		mcp := mcp
		bridge, err := NewBridge(mcp)
		if err != nil {
			return fmt.Errorf("failed to create bridge for %s: %v", mcp.Name, err)
		}
		bridges = append(bridges, bridge)

		mux := http.NewServeMux()
		mux.HandleFunc("/rpc", bridge.handleRPC)
		mux.HandleFunc("/sse", bridge.handleSSE)
		mux.HandleFunc("/health", bridge.handleHealth)
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"service": "MCPBridge",
				"mcp":     mcp.Name,
				"port":    mcp.Port,
				"endpoints": map[string]string{
					"rpc":    "/rpc (POST with application/json)",
					"sse":    "/sse (GET for Server-Sent Events)",
					"health": "/health (GET)",
				},
			})
		})

		go func(p int, n string) {
			log.Printf("Starting MCP %s on port %d", n, p)
			http.ListenAndServe(fmt.Sprintf(":%d", p), mux)
		}(mcp.Port, mcp.Name)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan

	log.Println("Shutting down MCPBridge...")
	for _, bridge := range bridges {
		bridge.Close()
	}
	removePIDFile()
	return nil
}

func main() {
	start := flag.Bool("start", false, "Start MCPBridge in background")
	stop := flag.Bool("stop", false, "Stop the running MCPBridge")
	flag.Parse()

	if *start && *stop {
		log.Fatal("Cannot use --start and --stop together")
	}

	if *start {
		args := flag.Args()
		configFile := "config.yaml"
		if len(args) > 0 {
			configFile = args[0]
		}
		if err := startDaemon(configFile); err != nil {
			log.Fatal(err)
		}
		return
	}

	if *stop {
		if err := stopDaemon(); err != nil {
			log.Fatal(err)
		}
		return
	}

	args := flag.Args()
	configFile := "config.yaml"
	if len(args) > 0 {
		configFile = args[0]
	}

	if err := runForeground(configFile); err != nil {
		log.Fatal(err)
	}
}
