package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"dipievil/mcpbridgego/internal/bridge"
	"dipievil/mcpbridgego/internal/output"

	"gopkg.in/yaml.v3"
)

// getPIDFile returns the path to the PID file.
func getPIDFile() string {
	pidFile := "/var/run/mcpbridgego.pid"
	if err := os.WriteFile(pidFile, []byte("test"), 0644); err == nil {
		os.Remove(pidFile)
		return pidFile
	}
	return filepath.Join(os.TempDir(), "mcpbridgego.pid")
}

// getLockFile returns the path to the lock file.
func getLockFile() string {
	return getPIDFile() + ".lock"
}

// acquireLock tries to acquire an exclusive lock for starting daemon.
func acquireLock() (*os.File, error) {
	lockFile := getLockFile()
	// O_CREATE|O_EXCL ensures only one process can create this file
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
	if err != nil {
		// Lock file already exists, another process is trying to start
		return nil, fmt.Errorf("MCPBridge startup is already in progress or another instance is running")
	}
	return f, nil
}

// releaseLock removes the lock file.
func releaseLock() error {
	lockFile := getLockFile()
	return os.Remove(lockFile)
}

// isProcessRunning checks if a process with the given PID is still running..
func isProcessRunning(pid int) bool {
	// First try the standard Unix signal method
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	defer process.Release()

	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}

	// Fallback: check /proc on Linux
	if _, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid))); err == nil {
		return true
	}

	return false
}

// savePID writes the current process ID to a file.
func savePID() error {
	pidFile := getPIDFile()
	pid := os.Getpid()
	return os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)
}

// readPID reads the PID from the PID file.
func readPID() (int, error) {
	pidFile := getPIDFile()
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

// removePIDFile removes the PID file.
func removePIDFile() error {
	pidFile := getPIDFile()
	return os.Remove(pidFile)
}

// startDaemon starts the app in background.
func startDaemon(configFile string) error {
	pidFile := getPIDFile()

	// Try to acquire startup lock to prevent race condition
	lockFile, err := acquireLock()
	if err != nil {
		// If lock file exists, either startup is in progress or daemon is running
		// Either way, we should return an error
		// Try to read PID to give a better error message
		if pid, err := readPID(); err == nil {
			if isProcessRunning(pid) {
				return fmt.Errorf("MCPBridge is already running (PID: %d)", pid)
			}
		}
		// Lock exists but can't determine if process is running
		return err
	}
	defer func() {
		if lockFile != nil {
			lockFile.Close()
		}
	}()
	defer func() {
		if lockFile != nil {
			lockFile.Close()
			releaseLock()
		}
	}()

	// Create new process to run in background
	cmd := exec.Command(os.Args[0], configFile)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCPBridge: %v", err)
	}

	// Write PID file immediately
	pidData := []byte(strconv.Itoa(cmd.Process.Pid))
	if err := os.WriteFile(pidFile, pidData, 0644); err != nil {
		log.Printf("Warning: could not write PID file: %v", err)
	}

	fmt.Printf("MCPBridge started in background (PID: %d)\n", cmd.Process.Pid)

	// Display startup info before exiting
	output.DisplayAgentCfgInfo()

	// Do NOT explicitly release lock - daemon will manage it
	// The lock file ownership transfers to daemon which will clean it up on exit
	//if lockFile != nil {
	//	lockFile.Close()
	//	releaseLock()
	//}

	// Exit the original process, leaving the new one in background
	os.Exit(0)
	return nil
}

// stopDaemon stops the running background process.
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

// runForeground runs the app in foreground mode.
func runForeground(configFile string) error {
	// If started via -start, the lock file is already created and held
	// by the parent -start process, which will exit soon
	// We just need to ensure it stays locked during daemon runtime and clean up on exit

	// Check if we're the daemon started by -start (lock file exists)
	// If so, we'll manage the lock on cleanup
	lockFileExists := false
	if _, err := os.Stat(getLockFile()); err == nil {
		lockFileExists = true
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("error reading config file: %v (%s)", err, configFile)
	}

	var tempConfig bridge.Config
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

	var config bridge.Config
	yaml.Unmarshal(data, &config)

	if err := savePID(); err != nil {
		log.Printf("Warning: could not write PID file: %v", err)
	}

	var bridges []*bridge.Bridge
	for _, mcp := range config.MCPS {
		b, err := bridge.NewBridge(mcp)
		if err != nil {
			return fmt.Errorf("failed to create bridge for %s: %v", mcp.Name, err)
		}
		bridges = append(bridges, b)

		mux := http.NewServeMux()
		mux.HandleFunc("/rpc", b.HandleRPC)
		mux.HandleFunc("/sse", b.HandleSSE)
		mux.HandleFunc("/health", b.HandleHealth)
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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan

	log.Println("Shutting down MCPBridge...")
	for _, b := range bridges {
		b.Close()
	}
	removePIDFile()

	// Clean up lock file if it was created by -start
	if lockFileExists {
		releaseLock()
	}
	return nil
}

func main() {

	var agent string
	var isFile bool
	var filePath string
	var start, stop, help bool

	osArgs := os.Args[1:]
	newArgs := []string{}
	for i := 0; i < len(osArgs); i++ {
		arg := osArgs[i]
		switch arg {
		case "-o", "--output":
			if i+1 < len(osArgs) && !strings.HasPrefix(osArgs[i+1], "-") {
				agent = osArgs[i+1]
				i++
			} else {
				agent = "generic"
			}
		case "-f", "--file":
			isFile = true
			if i+1 < len(osArgs) && !strings.HasPrefix(osArgs[i+1], "-") {
				filePath = osArgs[i+1]
				i++
			}
		case "-h", "--help":
			help = true
		case "-start":
			start = true
		case "-stop":
			stop = true
		default:
			newArgs = append(newArgs, arg)
		}
	}

	os.Args = append([]string{os.Args[0]}, newArgs...)

	if help {
		fmt.Println("MCPBridge - Model Context Protocol Bridge")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  mcpbridgego [options] [config_file]")
		fmt.Println()
		fmt.Println("Common options:")
		fmt.Println("  -start                   Start MCPBridge in background")
		fmt.Println("  -stop                    Stop the running MCPBridge")
		fmt.Println("  -h, --help               Show this help message")
		fmt.Println()
		fmt.Println("Output yml template:")
		fmt.Println("  -o, --output <agent>     Agent type: claude, copilot, generic (default: generic)")
		fmt.Println("  -f, --file [filename]    Output template to file (default: mcp.json)")
		fmt.Println()
		output.PrintOutputUsage()
		return
	}

	if agent != "" || isFile {
		if start || stop {
			log.Fatal("Cannot use --output/--file flags with --start or --stop")
		}

		outputCfg, err := output.ParseOutputConfig(agent, isFile, filePath)
		if err != nil {
			fmt.Printf("%sError:%s %v\n", output.ColorYellow, output.ColorReset, err)
			output.PrintOutputUsage()
			os.Exit(1)
		}

		if err := output.OutputMCPConfig(outputCfg); err != nil {
			fmt.Printf("%sError:%s %v\n", output.ColorYellow, output.ColorReset, err)
			os.Exit(1)
		}
		return
	}

	if start && stop {
		log.Fatal("Cannot use --start and --stop together")
	}

	if start {
		flag.Parse()
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

	if stop {
		if err := stopDaemon(); err != nil {
			log.Fatal(err)
		}
		return
	}

	flag.Parse()
	args := flag.Args()
	configFile := "config.yaml"
	if len(args) > 0 {
		configFile = args[0]
	}

	if err := runForeground(configFile); err != nil {
		log.Fatal(err)
	}
}
