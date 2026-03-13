package launcher

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const gfnFlatpakID = "com.nvidia.geforcenow"

func IsProcessRunning(nameSubstr string) bool {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}

	nameL := strings.ToLower(nameSubstr)

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name()[0] < '0' || entry.Name()[0] > '9' {
			continue
		}

		cmdline, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil {
			continue
		}

		parts := strings.Split(string(cmdline), "\x00")
		var cleanParts []string
		for _, p := range parts {
			if p != "" {
				cleanParts = append(cleanParts, p)
			}
		}
		if len(cleanParts) == 0 {
			continue
		}

		exeBase := strings.ToLower(filepath.Base(cleanParts[0]))
		cmdlineStr := strings.ToLower(strings.Join(cleanParts, " "))

		// Check if it's a helper process (zygote, etc.)
		isHelper := false
		for i := 1; i < len(cleanParts); i++ {
			if strings.HasPrefix(cleanParts[i], "--type=") {
				isHelper = true
				break
			}
		}

		// Tight match on the binary name or generic substring match for flatpaks
		var matches bool
		if nameL == "geforcenow" {
			// Special handling for GFN to ignore zygotes/helpers
			matches = (exeBase == "geforcenow" && !isHelper) ||
				(strings.Contains(cmdlineStr, "com.nvidia.geforcenow") &&
					(strings.Contains(exeBase, "flatpak") || strings.Contains(exeBase, "bwrap")) &&
					!strings.Contains(exeBase, "spawn"))
		} else {
			// Also check cmdlineStr to catch /proc/self/exe or other shells
			matches = strings.Contains(exeBase, nameL) || strings.Contains(cmdlineStr, nameL)
		}

		if matches &&
			!strings.Contains(cmdlineStr, "geforcenow-presence") &&
			!strings.Contains(cmdlineStr, "geforcenow-presence-dummies") {
			return true
		}
	}
	return false
}

// LaunchGFN starts the GeForce NOW Electron Flatpak.
func LaunchGFN() bool {
	if IsProcessRunning("GeForceNOW") {
		log.Println("💡 GeForce NOW is already running (launch skipped)")
		return true
	}

	log.Println("🚀 Launching GeForce NOW Flatpak...")
	cmd := exec.Command("flatpak", "run", gfnFlatpakID)
	if err := cmd.Start(); err != nil {
		log.Printf("❌ Failed to launch GeForce NOW: %v", err)
		return false
	}
	// Reap the process when it exits to avoid zombies
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("ℹ️ GeForce NOW process exited: %v", err)
		}
	}()
	return true
}

// LaunchDiscord starts Discord.
func LaunchDiscord() bool {
	if IsProcessRunning("Discord") {
		log.Println("💡 Discord is already running (launch skipped)")
		return true
	}

	// Try common Discord binary locations
	discordPaths := []string{
		"/usr/bin/Discord",
		"/usr/bin/discord",
		"/usr/local/bin/discord",
	}

	for _, p := range discordPaths {
		if _, err := os.Stat(p); err == nil {
			log.Println("🚀 Launching Discord...")
			cmd := exec.Command(p)
			if err := cmd.Start(); err != nil {
				log.Printf("❌ Failed to launch Discord: %v", err)
				return false
			}
			// Reap the process when it exits to avoid zombies
			go func() {
				if err := cmd.Wait(); err != nil {
					log.Printf("ℹ️ Discord process exited: %v", err)
				}
			}()
			return true
		}
	}

	// Try flatpak Discord
	cmd := exec.Command("flatpak", "run", "com.discordapp.Discord")
	if err := cmd.Start(); err != nil {
		log.Println("⚠️ Discord not found in standard locations or Flatpak")
		return false
	}
	// Reap the process when it exits to avoid zombies
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("ℹ️ Discord process exited: %v", err)
		}
	}()
	return true
}
