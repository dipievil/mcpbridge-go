package pidmanager

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// Manager handles PID file and lock file operations.
type Manager struct {
	pidFile  string
	lockFile string
}

// New creates a new PID manager.
func New() *Manager {
	pidFile := getPIDFilePath()
	return &Manager{
		pidFile:  pidFile,
		lockFile: pidFile + ".lock",
	}
}

// getPIDFilePath returns the path to the PID file.
func getPIDFilePath() string {
	pidFile := "/var/run/mcpbridgego.pid"
	if err := os.WriteFile(pidFile, []byte("test"), 0644); err == nil {
		os.Remove(pidFile)
		return pidFile
	}
	return filepath.Join(os.TempDir(), "mcpbridgego.pid")
}

// AcquireLock tries to acquire an exclusive advisory flock on the lock file.
// The returned file handle MUST be kept open as long as the lock should be held.
// The OS automatically releases the flock when the file is closed or the process exits,
// even on crashes via os.Exit or signals.
func (m *Manager) AcquireLock() (*os.File, error) {
	f, err := os.OpenFile(m.lockFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %v", err)
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("MCPBridge startup is already in progress or another instance is running")
	}

	return f, nil
}

// ReleaseLock removes the lock file.
func (m *Manager) ReleaseLock() error {
	return os.Remove(m.lockFile)
}

// IsProcessRunning checks if a process with the given PID is still running.
func (m *Manager) IsProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	defer process.Release()

	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}

	if _, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid))); err == nil {
		return true
	}

	return false
}

// RemoveProcess removes the PID file and releases the lock.
func (m *Manager) RemoveProcess() error {
	err := m.removePID()
	if err != nil {
		return err
	}
	return m.ReleaseLock()
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

// removePID removes the PID file
func (m *Manager) removePID() error {
	return os.Remove(m.pidFile)
}

// LockFileExists checks if lock file exists
func (m *Manager) LockFileExists() bool {
	_, err := os.Stat(m.lockFile)
	return err == nil
}

// GetPIDFile Returns the path to the PID file.
func (m *Manager) GetPIDFile() string {
	return m.pidFile
}

// CleanupOrphanedLock removes the lock file if no process currently holds the advisory flock.
// With flock, no PID check is needed: the OS automatically releases the lock when
// the holding process exits for any reason (normal exit, crash, SIGKILL, etc.).
func (m *Manager) CleanupOrphanedLock() error {
	if !m.LockFileExists() {
		return nil
	}

	f, err := os.OpenFile(m.lockFile, os.O_RDWR, 0644)
	if err != nil {
		return os.Remove(m.lockFile)
	}

	flockErr := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	f.Close()

	if flockErr != nil {
		return nil
	}

	return os.Remove(m.lockFile)
}
