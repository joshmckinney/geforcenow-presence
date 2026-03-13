package launcher

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// DummyLauncher manages "fake" game processes to trick Discord detection.
type DummyLauncher struct {
	dummyDir string
	cmd      *exec.Cmd
}

// NewDummyLauncher creates a new DummyLauncher.
func NewDummyLauncher() *DummyLauncher {
	dummyDir := filepath.Join(os.TempDir(), "geforcenow-presence-dummies")
	return &DummyLauncher{dummyDir: dummyDir}
}

// Start spawns a dummy process with the given executable name.
func (l *DummyLauncher) Start(exeName string) (int, error) {
	l.Stop() // Ensure any previous dummy is stopped

	dummyPath := filepath.Join(l.dummyDir, exeName)

	if err := os.MkdirAll(filepath.Dir(dummyPath), 0755); err != nil {
		return 0, fmt.Errorf("failed to create dummy dir: %w", err)
	}

	// Copy a simple "no-op" file if it doesn't exist
	// We'll use /usr/bin/tail -f /dev/null as a long-running no-op process
	// But we need the FILE to be named exeName.
	// On Linux, we can just symlink tail to the target name.
	tailPath, err := exec.LookPath("tail")
	if err != nil {
		// Fallback to sleep if tail is not found
		tailPath, err = exec.LookPath("sleep")
		if err != nil {
			return 0, fmt.Errorf("could not find tail or sleep: %w", err)
		}
	}

	// Remove old dummy if it exists
	_ = os.Remove(dummyPath)

	// Create a copy of the binary to the dummy path
	// Symlinks might be followed by Discord and show original name?
	// Let's copy to be safe.
	if err := copyFile(tailPath, dummyPath); err != nil {
		return 0, fmt.Errorf("failed to copy dummy binary: %w", err)
	}

	// Make it executable
	if err := os.Chmod(dummyPath, 0755); err != nil {
		return 0, fmt.Errorf("failed to chmod dummy binary: %w", err)
	}

	// Run it with -f /dev/null (for tail) or a very long time (for sleep)
	var cmd *exec.Cmd
	if filepath.Base(tailPath) == "tail" {
		cmd = exec.Command(dummyPath, "-f", "/dev/null")
	} else {
		cmd = exec.Command(dummyPath, "infinity")
	}

	// Start the process in its own group so it doesn't get signals meant for us
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start dummy process: %w", err)
	}

	l.cmd = cmd
	log.Printf("🚀 Started dummy process %s (PID %d)", exeName, cmd.Process.Pid)
	return cmd.Process.Pid, nil
}

// Stop terminates the currently running dummy process.
func (l *DummyLauncher) Stop() {
	if l.cmd != nil && l.cmd.Process != nil {
		log.Printf("🛑 Stopping dummy process (PID %d)", l.cmd.Process.Pid)
		// Try to kill the whole process group
		pgid, err := syscall.Getpgid(l.cmd.Process.Pid)
		if err == nil {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			_ = l.cmd.Process.Kill()
		}
		_ = l.cmd.Wait()
		l.cmd = nil
	}
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
