package main

import (
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
	"dipievil/mcpbridgego/internal/logger"
	"dipievil/mcpbridgego/internal/output"
	"dipievil/mcpbridgego/internal/pidmanager"
)

var buildVersion = "dev"

// startDaemon starts the app in background.
func startDaemon(pm *pidmanager.Manager) error {

	pm.CleanupOrphanedLock()

	lockFile, err := pm.AcquireLock()
	if err != nil {

		if pid, err := pm.ReadPID(); err == nil && pm.IsProcessRunning(pid) {
			return fmt.Errorf("MCPBridge is already running (PID: %d)", pid)
		}
		return fmt.Errorf("MCPBridge startup is already in progress or another instance is running")
	}

	if lockFile != nil {
		defer lockFile.Close()
	}

	cmd := exec.Command(os.Args[0])
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		pm.ReleaseLock()
		return fmt.Errorf("failed to start MCPBridge: %v", err)
	}

	pidData := []byte(strconv.Itoa(cmd.Process.Pid))
	if err := os.WriteFile(pm.GetPIDFile(), pidData, 0644); err != nil {
		log.Printf("Warning: could not write PID file: %v", err)
	}

	fmt.Printf("MCPBridge started in background (PID: %d)\n", cmd.Process.Pid)

	output.DisplayAgentCfgInfo()

	lockFile.Close()

	os.Exit(0)
	return nil
}

// getRunningPID returns the PID if a process is running, or an error
func getRunningPID(pm *pidmanager.Manager) (int, error) {
	pid, err := pm.ReadPID()
	if err != nil {
		return 0, fmt.Errorf("MCPBridge is not running (no PID file found)")
	}

	if !pm.IsProcessRunning(pid) {
		if err := pm.RemoveProcess(); err != nil {
			log.Printf("Warning: failed to remove MCPBridge process: %v", err)
		}
		return 0, fmt.Errorf("MCPBridge is not running (PID %d not found)", pid)
	}

	return pid, nil
}

// stopDaemon stops the running background process.
func stopDaemon(pm *pidmanager.Manager) error {
	pid, err := getRunningPID(pm)
	if err != nil {
		return err
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %v", err)
	}
	defer process.Release()

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to stop MCPBridge: %v", err)
	}

	if err := pm.RemoveProcess(); err != nil {
		return fmt.Errorf("failed to remove MCPBridge process: %v", err)
	}

	fmt.Printf("MCPBridge stopped (PID: %d)\n", pid)
	return nil
}

// checkStatus checks if MCPBridge is running.
func checkStatus(pm *pidmanager.Manager) error {
	pid, err := getRunningPID(pm)
	if err != nil {
		fmt.Println("MCPBridge is not running")
		return nil
	}

	fmt.Printf("MCPBridge is running (PID: %d)\n", pid)
	return nil
}

// validateConfig validates the config file.
func validateConfig() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	if err := config.Validate(cfg); err != nil {
		return fmt.Errorf("config validation failed: %v", err)
	}

	fmt.Println("Config validated successfully")
	return nil
}

// parseArgs parses command-line arguments and returns an AppArgs struct.
func parseArgs(osArgs []string) (config.AppArgs, error) {

	appArgs := config.AppArgs{}

	for i := 0; i < len(osArgs); i++ {
		arg := osArgs[i]
		switch arg {
		case "-o", "--output":
			if i+1 < len(osArgs) && !strings.HasPrefix(osArgs[i+1], "-") {
				appArgs.AgentName = osArgs[i+1]
				appArgs.OutputConfig = true
				i++
			} else {
				appArgs.AgentName = "generic"
				appArgs.OutputConfig = true
			}
		case "-f", "--file":
			appArgs.OutputAsFile = true
			if i+1 < len(osArgs) && !strings.HasPrefix(osArgs[i+1], "-") {
				appArgs.FilePath = osArgs[i+1]
				i++
			}
		case "-h", "--help":
			appArgs.ShowHelp = true
		case "-c", "--config":
			appArgs.ValidateConfig = true
		case "-s", "--start":
			appArgs.RunStart = true
		case "-t", "--stop":
			appArgs.RunStop = true
		case "--status":
			appArgs.GetStatus = true
		case "-r", "--run":
			appArgs.RunForeground = true
		default:
			return config.AppArgs{}, fmt.Errorf("unknown argument: %s", arg)
		}
	}

	if err := validateConflictArgs(appArgs); err != nil {
		return config.AppArgs{}, err
	}

	return appArgs, nil
}

// validateConflictArgs checks for conflicting command-line arguments.
func validateConflictArgs(appArgs config.AppArgs) error {
	if appArgs.OutputConfig || appArgs.AgentName != "" || appArgs.OutputAsFile {
		if appArgs.RunStart || appArgs.RunStop || appArgs.RunForeground || appArgs.GetStatus || appArgs.ValidateConfig {
			return fmt.Errorf("cannot use --output/--file flags with --start, --stop, --run, --status or --config")
		}
	}

	if appArgs.RunStart && appArgs.RunStop {
		return fmt.Errorf("cannot use --start and --stop together")
	}

	if appArgs.RunStart && appArgs.GetStatus {
		return fmt.Errorf("cannot use --start and --status together")
	}

	if appArgs.RunStop && appArgs.GetStatus {
		return fmt.Errorf("cannot use --stop and --status together")
	}

	if appArgs.ValidateConfig && (appArgs.RunStart || appArgs.RunStop || appArgs.RunForeground || appArgs.GetStatus) {
		return fmt.Errorf("cannot use --config with --start, --stop, --run, or --status")
	}

	return nil
}

func main() {

	osArgs := os.Args[1:]

	if len(osArgs) == 0 {
		output.PrintMainHelp()
		return
	}

	appArgs, err := parseArgs(osArgs)
	if err != nil {
		log.Fatal(err)
	}

	if appArgs.ValidateConfig {
		if err := validateConfig(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if appArgs.OutputConfig {
		if err := outputConfig(appArgs); err != nil {
			log.Fatal(err)
		}
		return
	}
	pm := pidmanager.New()

	if appArgs.GetStatus {
		if err := checkStatus(pm); err != nil {
			log.Fatal(err)
		}
		return
	}

	if appArgs.RunStart {
		if err := startDaemon(pm); err != nil {
			log.Fatal(err)
		}
		return
	}

	if appArgs.RunStop {
		if err := stopDaemon(pm); err != nil {
			log.Fatal(err)
		}
		return
	}

	if appArgs.RunForeground {
		if err := runForeground(pm); err != nil {
			log.Fatal(err)
		}
		return
	}

	log.Println("Showing help message")
	output.PrintMainHelp()
}

func outputConfig(appArgs config.AppArgs) error {
	outputCfg, err := output.ParseOutputConfig(appArgs.AgentName, appArgs.OutputAsFile, appArgs.FilePath)
	if err != nil {
		output.PrintOutputUsage()
		return fmt.Errorf("%sError:%s %v", output.ColorYellow, output.ColorReset, err)
	}

	if err := output.OutputMCPConfig(outputCfg); err != nil {
		return fmt.Errorf("%sError:%s %v", output.ColorYellow, output.ColorReset, err)
	}

	return nil
}

// runForeground runs the app in foreground mode.
func runForeground(pm *pidmanager.Manager) error {

	lockFile, err := pm.AcquireLock()
	if err != nil {
		return fmt.Errorf("MCPBridge is already running: %v", err)
	}
	defer lockFile.Close()

	fileLogger := logger.InitFileLogger("log.txt")
	defer fileLogger.Close()

	cfg, err := config.LoadConfig()
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
		mux.HandleFunc("/", b.HandleSSE)

		go func(port int, name string) {
			log.Printf("Starting MCP %s on port %d", name, port)
			if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
				log.Fatalf("Failed to start HTTP server for MCP %s on port %d: %v", name, port, err)
			}
		}(mcp.Port, mcp.Name)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan

	log.Println("Shutting down MCPBridge...")
	for _, b := range bridges {
		b.Close()
	}
	if err := pm.RemoveProcess(); err != nil {
		log.Printf("Warning: failed to remove MCPBridge process: %v", err)
	}
	return nil
}
