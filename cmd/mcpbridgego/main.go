package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type MCPConfig struct {
	Name    string   `yaml:"name"`
	Port    int      `yaml:"port"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
	EnvFile string   `yaml:"env_file"`
}

type Config struct {
	MCPS []MCPConfig `yaml:"mcps"`
}

type Bridge struct {
	config MCPConfig
	mu     sync.Mutex
	stdin  io.WriteCloser
	cmd    *exec.Cmd
}

func (b *Bridge) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	b.mu.Lock()

	envs, _ := godotenv.Read(b.config.EnvFile)

	b.cmd = exec.Command(b.config.Command, b.config.Args...)
	b.cmd.Env = os.Environ()
	for k, v := range envs {
		b.cmd.Env = append(b.cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	b.stdin, _ = b.cmd.StdinPipe()
	stdout, _ := b.cmd.StdoutPipe()
	b.mu.Unlock()

	if err := b.cmd.Start(); err != nil {
		log.Printf("Error starting %s: %v", b.config.Name, err)
		return
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Fprintf(w, "data: %s\n\n", scanner.Text())
			w.(http.Flusher).Flush()
		}
	}()

	<-r.Context().Done()
	b.mu.Lock()
	if b.cmd != nil && b.cmd.Process != nil {
		b.cmd.Process.Kill()
	}
	b.mu.Unlock()
}

func (b *Bridge) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	body, _ := io.ReadAll(r.Body)
	b.mu.Lock()
	if b.stdin != nil {
		b.stdin.Write(body)
		b.stdin.Write([]byte("\n"))
	}
	b.mu.Unlock()
	w.WriteHeader(http.StatusAccepted)
}

func main() {

	configFile := "config.yaml"
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatal("Error reading config file: ", err)
	}

	var tempConfig Config
	yaml.Unmarshal(data, &tempConfig)
	for _, mcp := range tempConfig.MCPS {
		if _, err := os.Stat(mcp.EnvFile); os.IsNotExist(err) {
			log.Fatalf("Env file %s for MCP %s does not exist. Check yaml file.", mcp.EnvFile, mcp.Name)
		}
	}

	for _, mcp := range tempConfig.MCPS {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", mcp.Port))
		if err != nil {
			log.Fatalf("Port %d for MCP %s is not available. Check if another process is using it.", mcp.Port, mcp.Name)
		}
		ln.Close()
	}

	for _, mcp := range tempConfig.MCPS {
		if _, err := exec.LookPath(mcp.Command); err != nil {
			log.Fatalf("Command %s for MCP %s not found in PATH. Check yaml file.", mcp.Command, mcp.Name)
		}
	}

	var config Config
	yaml.Unmarshal(data, &config)

	for _, mcp := range config.MCPS {
		mcp := mcp
		bridge := &Bridge{config: mcp}

		mux := http.NewServeMux()
		mux.HandleFunc("/sse", bridge.handleSSE)
		mux.HandleFunc("/messages", bridge.handleMessages)

		go func(p int, name string) {
			log.Printf("Starting MCP %s on port %d", name, p)
			http.ListenAndServe(fmt.Sprintf(":%d", p), mux)
		}(mcp.Port, mcp.Name)
	}

	select {}
}
