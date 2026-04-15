# MCPBridge Project Instructions

MCPBridge is a Model Context Protocol (MCP) HTTP bridge for exposing MCPs to AI agents over the network. This instruction file captures project-specific patterns and conventions.

## Architecture Overview

- **Daemon Mode**: Background process with PID file tracking and signal handling
- **Configuration**: YAML-based, parsed dynamically with support for multiple MCPs
- **Protocol**: JSON-RPC 2.0 over HTTP with Server-Sent Events (SSE) support
- **Isolation**: Each MCP runs as separate child process with environment isolation

## Critical Patterns

### 1. Daemon Process Management

**Race Condition Prevention:**
- Always use atomic file operations (`O_CREATE|O_EXCL`) for startup synchronization
- Lock file pattern: `{pidfile}.lock` prevents concurrent daemon starts
- Lock acquisition must happen BEFORE spawning the daemon process
- Lock file must be cleaned up on graceful shutdown (SIGTERM handler)

```go
// Correct: Lock acquired first, held until daemon exits
lockFile, err := acquireLock()  // atomic operation
if err != nil {
    return fmt.Errorf("MCPBridge startup is already in progress...")
}
cmd := exec.Command(...)  // Safe to spawn
```

**PID File Handling:**
- Store PID file in `/var/run/mcpbridgego.pid` with fallback to `/tmp/`
- Write PID immediately after process spawn
- Remove PID file on clean shutdown
- Handle stale PID files (check if process still exists)

### 2. Configuration Parsing

**YAML Structure:**
- Use struct tags without spaces: `yaml:"field_name,omitempty"` (not `"field, omitempty"`)
- All MCPs defined under `mcps:` array
- Each MCP has: `name`, `port`, `command`, `args` (required); `env_file`, `env_vars`, `dir` (optional)

**Dynamic Generation Rules:**
- Never hardcode configuration values in code
- Always parse from config file at startup
- Generate output/display values dynamically from actual configuration
- When displaying MCP URLs, detect local IP instead of using `localhost`

```go
// Correct: Generate from config
config, err := GenerateDynamicMCPConfig(configFile)
servers["mcp_name"] = map[string]interface{}{
    "url": fmt.Sprintf("http://%s:%d", localIP, port),
}

// Wrong: Hardcoded templates
"example_server": {"url": "http://localhost:3000"}
```

**Optional Fields:**
- `env_file` is optional - only validate if specified
- `env_vars` can be used as alternative or supplement to `env_file`
- `dir` (working directory) is optional

```go
// Correct: Check if field is specified
if mcp.EnvFile != "" {
    if _, err := os.Stat(mcp.EnvFile); err != nil {
        // File doesn't exist - error only if specified
    }
}
```

### 3. Error Handling

**Pattern:**
- Return errors as last return value
- Wrap errors with context: `fmt.Errorf("operation: %v", err)`
- Use early exit (return) - no silent failures
- Provide actionable error messages mentioning config file location

```go
// Correct: Context-wrapped error, mentions config
if _, err := os.Stat(mcp.Command); err != nil {
    return fmt.Errorf("command %s for MCP %s not found in PATH. Check yaml file", 
        mcp.Command, mcp.Name)
}
```

### 4. Signal Handling & Cleanup

**Shutdown Sequence:**
1. Receive SIGTERM or SIGINT
2. Signal all child MCP processes
3. Close all bridge connections
4. Remove PID file
5. Remove lock file
6. Exit cleanly

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
<-sigChan

// Cleanup here before exit
removePIDFile()
releaseLock()
```

### 5. Process Startup (-start flag)

**Behavior:**
- Fork new process in background
- Parent `-start` process exits after child spawns
- Child (daemon) manages lock file and stays running
- Cannot use `defer` for cleanup before `os.Exit()` (deferred functions don't run)
- Explicitly clean up locks before exiting parent

```go
// Correct: Lock cleaned up before exit
fmt.Printf("MCPBridge started in background (PID: %d)\n", cmd.Process.Pid)
output.DisplayAgentCfgInfo(configFile) // Dynamic output
// Clean up parent's reference to lock before exit
if lockFile != nil {
    lockFile.Close()
    // Don't release - daemon inherits lock responsibility
}
os.Exit(0)  // Deferred cleanup won't run
```

## Testing Conventions

**Build-First Pattern:**
- Always build binary before testing
- Test with actual compiled binary, not just syntax
- Verify all flags work: `-start`, `-stop`, config file changes

**Test Scenarios:**
1. Daemon startup without env_file (optional fields)
2. Multiple rapid `-start` calls (lock prevents duplicates)
3. Stop after start (verify lock cleanup)
4. Restart after stop (lock file properly removed)
5. Dynamic configuration display (not hardcoded)

**Run Order:**
```bash
go build -o mcpbridgego ./cmd/mcpbridgego/
go test ./...
./mcpbridgego -start config.yaml  # Integration test
./mcpbridgego -stop
```

## Command Flags

- `-start <config>`: Start MCPBridge in background with config file
- `-stop`: Stop running MCPBridge daemon
- `-o, --output <agent>`: Output template for agent (claude, copilot, generic)
- `-f, --file [path]`: Save configuration to file
- `-h, --help`: Show help

## JSON-RPC 2.0 Message Format

```json
{
  "jsonrpc": "2.0",
  "method": "string",
  "params": {},
  "id": "unique identifier"
}

Response:
{
  "jsonrpc": "2.0",
  "result": {},
  "id": "same as request"
}

Error:
{
  "jsonrpc": "2.0",
  "error": {"code": -32603, "message": "Internal error"},
  "id": "same as request"
}
```

## HTTP Endpoints

- `POST /rpc`: JSON-RPC method calls
- `GET /sse`: Server-Sent Events stream
- `GET /health`: Health check
- `GET /`: Service info with available endpoints (JSON)

**CORS:** All endpoints respond to requests from any origin (Access-Control-Allow-Origin: *)

## Commit Message Pattern

Follow Conventional Commits specification (already enforced by git-conventional-commits.instructions.md):
- `fix: prevent duplicate daemon instances with atomic lock file`
- `feat: generate MCP configuration dynamically from config file`
- `refactor: move bridge methods to internal/bridge package`

## Debugging Tips

**Daemon not visible?**
Check if another instance is holding the lock file:
```bash
ls -la /tmp/mcpbridgego.pid*
ps aux | grep mcpbridgego
```

**Lock file persists after stop?**
Verify the daemon cleanup code calls `releaseLock()` in the shutdown handler.

**Configuration not showing actual MCPs?**
Ensure `DisplayAgentCfgInfo(configFile)` is called with the config file path, not hardcoded template.

**YAML parsing errors?**
Check struct tags for spaces: `yaml:"env_file,omitempty"` ✓ vs `yaml:"env_file, omitempty"` ✗

## Related Documentation

- Conventional Commits: see `git-conventional-commits.instructions.md`
- Project structure checklist: see `project-structure.instructions.md`
- Build Make tasks: see `Makefile` and `makefile.instructions.md`

## Overall Conventions
 - Never add comments that explain "what" the code is doing inside any function.
 Ex:
 ```go
// Get the local IP for the server
localIP := getLocalIPForServer()
```
 - Always write code that is self-explanatory and clear enough to understand the "what" without needing comments.
 - Use comments only to explain "why" certain decisions were made, especially if they are non-obvious or could be misunderstood.
 - Avoid DRY violations by not repeating code patterns that can be abstracted into functions or methods.
 - Always follow the established patterns in the project for consistency, especially around error handling, configuration parsing, and process management.
 - Always confirm before committing.

 