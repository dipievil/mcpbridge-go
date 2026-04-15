package pidmanager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPIDFileHandling(t *testing.T) {
	pm := New()
	pidFile := pm.GetPIDFile()

	// Clean up before test
	defer os.Remove(pidFile)

	// Save PID
	if err := pm.SavePID(); err != nil {
		t.Fatalf("failed to save PID: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(pidFile); err != nil {
		t.Errorf("PID file not created at %s", pidFile)
	}

	// Read PID
	pid, err := pm.ReadPID()
	if err != nil {
		t.Fatalf("failed to read PID: %v", err)
	}

	expectedPID := os.Getpid()
	if pid != expectedPID {
		t.Errorf("expected PID %d, got %d", expectedPID, pid)
	}

	// Cleanup
	if err := pm.RemovePID(); err != nil {
		t.Fatalf("failed to remove PID file: %v", err)
	}

	// Verify file was removed
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Errorf("PID file still exists after removal")
	}
}

func TestGetPIDFile(t *testing.T) {
	pm := New()
	pidFile := pm.GetPIDFile()

	if pidFile == "" {
		t.Error("PID file path is empty")
	}

	// Should be either /var/run/mcpbridgego.pid or in temp directory
	if pidFile != "/var/run/mcpbridgego.pid" {
		tempDir := os.TempDir()
		if !filepath.HasPrefix(pidFile, tempDir) {
			t.Errorf("PID file should be in /var/run or temp directory, got %s", pidFile)
		}
	}
}

func TestIsProcessRunning(t *testing.T) {
	pm := New()

	// Current process should be running
	currentPID := os.Getpid()
	if !pm.IsProcessRunning(currentPID) {
		t.Errorf("current process (PID %d) should be running", currentPID)
	}

	// Invalid PID should not be running
	invalidPID := 99999999
	if pm.IsProcessRunning(invalidPID) {
		t.Errorf("invalid PID %d should not be running", invalidPID)
	}
}

func TestAcquireAndReleaseLock(t *testing.T) {
	pm := New()

	// Clean up before test
	os.Remove(pm.lockFile)
	defer os.Remove(pm.lockFile)

	// Acquire lock
	lockFile, err := pm.AcquireLock()
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}
	defer lockFile.Close()

	// Verify lock file exists
	if !pm.LockFileExists() {
		t.Error("lock file should exist after acquiring lock")
	}

	// Try to acquire lock again - should fail
	_, err = pm.AcquireLock()
	if err == nil {
		t.Error("should not be able to acquire lock twice")
	}

	// Release lock
	lockFile.Close()
	if err := pm.ReleaseLock(); err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	// Verify lock file is removed
	if pm.LockFileExists() {
		t.Error("lock file should not exist after release")
	}
}


func TestReadPIDError(t *testing.T) {
	pm := New()
	pidFile := pm.GetPIDFile()

	// Ensure PID file doesn't exist
	os.Remove(pidFile)

	// Try to read non-existent PID file
	_, err := pm.ReadPID()
	if err == nil {
		t.Error("should fail to read non-existent PID file")
	}
}

func TestLockFileExists(t *testing.T) {
	pm := New()
	defer os.Remove(pm.lockFile)

	// Initially, lock file should not exist
	if pm.LockFileExists() {
		t.Error("lock file should not exist initially")
	}

	// Create lock file
	lockFile, err := pm.AcquireLock()
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}
	defer lockFile.Close()

	// Lock file should exist
	if !pm.LockFileExists() {
		t.Error("lock file should exist after acquiring lock")
	}
}
