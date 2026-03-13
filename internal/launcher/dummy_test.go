package launcher

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestDummyLauncher(t *testing.T) {
	l := NewDummyLauncher()
	exeName := "test-game-dummy"

	pid, err := l.Start(exeName)
	if err != nil {
		t.Fatalf("Failed to start dummy: %v", err)
	}

	if pid <= 0 {
		t.Errorf("Invalid PID: %d", pid)
	}

	dummyPath := filepath.Join(l.dummyDir, exeName)
	if _, err := os.Stat(dummyPath); os.IsNotExist(err) {
		t.Errorf("Dummy executable not found at %s", dummyPath)
	}

	// Wait a moment for process to settle
	time.Sleep(100 * time.Millisecond)

	// Check if process is alive
	process, err := os.FindProcess(pid)
	if err != nil {
		t.Fatalf("Failed to find process: %v", err)
	}

	// On Unix, FindProcess always succeeds, we need to send signal 0
	err = process.Signal(os.Signal(syscall.Signal(0)))
	if err != nil {
		t.Errorf("Process %d is not running: %v", pid, err)
	}

	l.Stop()

	// Verify process is stopped
	time.Sleep(100 * time.Millisecond)
	err = process.Signal(os.Signal(syscall.Signal(0)))
	if err == nil {
		t.Errorf("Process %d is still running after Stop()", pid)
	}
}

// Fixed version of the test for better reliability
func TestDummyLauncherReliable(t *testing.T) {
	l := NewDummyLauncher()
	exeName := "test-game-logic"

	pid, err := l.Start(exeName)
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Check if dummy file exists
	dummyPath := filepath.Join(l.dummyDir, exeName)
	if _, err := os.Stat(dummyPath); err != nil {
		t.Errorf("Dummy file missing: %v", err)
	}

	// Check process name via /proc
	commBytes, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "comm"))
	if err == nil {
		comm := strings.TrimSpace(string(commBytes))
		if comm != exeName {
			t.Errorf("Expected comm %s, got %s", exeName, comm)
		}
	}

	l.Stop()
}

func TestDummyLauncherNested(t *testing.T) {
	l := NewDummyLauncher()
	exeName := "folder/test-game-nested.exe"

	_, err := l.Start(exeName)
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Check if dummy file exists
	dummyPath := filepath.Join(l.dummyDir, exeName)
	if _, err := os.Stat(dummyPath); err != nil {
		t.Errorf("Nested dummy file missing: %v", err)
	}

	l.Stop()
}
