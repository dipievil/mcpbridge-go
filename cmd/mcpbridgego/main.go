package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"dipievil/mcpbridgego/internal/bridge"
	"dipievil/mcpbridgego/internal/config"
	"dipievil/mcpbridgego/internal/output"
	"dipievil/mcpbridgego/internal/pidmanager"
)

// startDaemon starts the app in background.
func startDaemon(configFile string, pm *pidmanager.Manager) error {
	// Clean up any orphaned lock file from a previous failed startup
	pm.CleanupOrphanedLock()

	// Try to acquire startup lock to prevent race condition
	lockFile, err := pm.AcquireLock()
	if err != nil {
		// If lock file exists, either startup is in progress or daemon is running
		// Try to read PID to give a better error message
		if pid, err := pm.ReadPID(); err == nil {
			if pm.IsProcessRunning(pid) {
				return fmt.Errorf("MCPBridge is already running (PID: %d)", pid)
			}
		}
		// Lock file exists but process is not running - retry after cleanup
		pm.CleanupOrphanedLock()
		lockFile, err = pm.AcquireLock()
		if err != nil {
			return fmt.Errorf("MCPBridge startup is already in progress or another instance is running")
		}
	}

	if lockFile != nil {
		defer lockFile.Close()
	}

	// Create new process to run in background
	cmd := exec.Command(os.Args[0], configFile)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		pm.ReleaseLock()
		return fmt.Errorf("failed to start MCPBridge: %v", err)
	}

	// Write PID file immediately
	pidData := []byte(strconv.Itoa(cmd.Process.Pid))
	if err := os.WriteFile(pm.GetPIDFile(), pidData, 0644); err != nil {
		log.Printf("Warning: could not write PID file: %v", err)
	}

	fmt.Printf("MCPBridge started in background (PID: %d)\n", cmd.Process.Pid)

	// Display startup info before exiting
	output.DisplayAgentCfgInfo(configFile)

	// Lock file is left open and will be cleaned up by the daemon on shutdown
	// Daemon is now responsible for: managing the lock, running MCPs, and cleanup
	lockFile.Close()

	// Exit the original process, leaving the new one in background
	os.Exit(0)
	return nil
}

// stopDaemon stops the running background process.
func stopDaemon(pm *pidmanager.Manager) error {
	pid, err := pm.ReadPID()
	if err != nil {
		return fmt.Errorf("MCPBridge is not running (no PID file found)")
	}

	if !pm.IsProcessRunning(pid) {
		pm.RemovePID()
		// Clean up any orphaned lock file
		pm.ReleaseLock()
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

	pm.RemovePID()
	// Clean up lock file
	pm.ReleaseLock()
	fmt.Printf("MCPBridge stopped (PID: %d)\n", pid)
	return nil
}

// runForeground runs the app in foreground mode.
func runForeground(configFile string, pm *pidmanager.Manager) error {
	// Load and validate config
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return err
	}

	if err := config.Validate(cfg); err != nil {
		return err
	}

	if err := pm.SavePID(); err != nil {
		log.Printf("Warning: could not write PID file: %v", err)
	}

	var bridges []*bridge.Bridge
	for _, mcp := range cfg.MCPS {
		b, err := bridge.NewBridge(mcp)
		if err != nil {
			return fmt.Errorf("failed to create bridge for %s: %v", mcp.Name, err)
		}
		bridges = append(bridges, b)

		mux := http.NewServeMux()
		mux.HandleFunc("/rpc", b.HandleRPC)
		mux.HandleFunc("/sse", b.HandleSSE)
		mux.HandleFunc("/health", b.HandleHealth)
		// Root route defaults to SSE (MCP standard protocol)
		mux.HandleFunc("/", b.HandleSSE)

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
	pm.RemovePID()
	// Always clean up lock file when daemon exits
	pm.ReleaseLock()
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
		output.PrintMainHelp()
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

	pm := pidmanager.New()

	if start {
		flag.Parse()
		args := flag.Args()
		configFile := "config.yaml"
		if len(args) > 0 {
			configFile = args[0]
		}
		if err := startDaemon(configFile, pm); err != nil {
			log.Fatal(err)
		}
		return
	}

	if stop {
		if err := stopDaemon(pm); err != nil {
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

	if err := runForeground(configFile, pm); err != nil {
		log.Fatal(err)
	}
}
