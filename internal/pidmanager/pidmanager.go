package pidmanager

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// Manager handles PID file and lock file operations
type Manager struct {
	pidFile  string
	lockFile string
}

// New creates a new PID manager
func New() *Manager {
	pidFile := getPIDFilePath()
	return &Manager{
		pidFile:  pidFile,
		lockFile: pidFile + ".lock",
	}
}

// getPIDFilePath returns the path to the PID file
func getPIDFilePath() string {
	pidFile := "/var/run/mcpbridgego.pid"
	if err := os.WriteFile(pidFile, []byte("test"), 0644); err == nil {
		os.Remove(pidFile)
		return pidFile
	}
	return filepath.Join(os.TempDir(), "mcpbridgego.pid")
}

// AcquireLock tries to acquire an exclusive lock for daemon startup
func (m *Manager) AcquireLock() (*os.File, error) {
	f, err := os.OpenFile(m.lockFile, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
	if err != nil {
		return nil, fmt.Errorf("MCPBridge startup is already in progress or another instance is running")
	}
	return f, nil
}

// ReleaseLock removes the lock file
func (m *Manager) ReleaseLock() error {
	return os.Remove(m.lockFile)
}

// IsProcessRunning checks if a process with the given PID is still running
func (m *Manager) IsProcessRunning(pid int) bool {
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

// SavePID writes the current process ID to the PID file
func (m *Manager) SavePID() error {
	pid := os.Getpid()
	return os.WriteFile(m.pidFile, []byte(strconv.Itoa(pid)), 0644)
}

// ReadPID reads the PID from the PID file
func (m *Manager) ReadPID() (int, error) {
	data, err := os.ReadFile(m.pidFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

// RemovePID removes the PID file
func (m *Manager) RemovePID() error {
	return os.Remove(m.pidFile)
}

// LockFileExists checks if lock file exists
func (m *Manager) LockFileExists() bool {
	_, err := os.Stat(m.lockFile)
	return err == nil
}

// GetPIDFile returns the path to the PID file
func (m *Manager) GetPIDFile() string {
	return m.pidFile
}

// CleanupOrphanedLock removes lock file if the process in PID file is not running
func (m *Manager) CleanupOrphanedLock() error {
	// Only cleanup if lock file exists
	if !m.LockFileExists() {
		return nil
	}

	// Try to read the PID
	pid, err := m.ReadPID()
	if err != nil {
		// PID file doesn't exist or can't be read - lock is orphaned
		return m.ReleaseLock()
	}

	// Check if process is still running
	if !m.IsProcessRunning(pid) {
		// Process is not running - lock is orphaned
		return m.ReleaseLock()
	}

	// Process is still running - don't cleanup
	return nil
}
