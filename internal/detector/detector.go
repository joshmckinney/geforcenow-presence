package detector

import (
	"bufio"
	"embed"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	gfnProcessName = "GeForceNOW"
	extensionUUID  = "window-title-server@geforcenow-presence"
)

var (
	trademarkRe = regexp.MustCompile(`[®™]`)
)

//go:embed extension.js metadata.json
var extensionFiles embed.FS

// Detector detects the GeForce NOW Flatpak process and game being played.
type Detector struct {
	lastMethod       string
	extensionChecked bool
	extensionReady   bool
}

// New creates a new Detector. Ensures the GNOME Shell extension is installed.
func New() *Detector {
	d := &Detector{}
	d.ensureExtension()
	return d
}

// IsGFNRunning checks if the GeForce NOW Electron Flatpak is running.
func (d *Detector) IsGFNRunning() bool {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name()[0] < '0' || entry.Name()[0] > '9' {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(string(cmdline)), strings.ToLower(gfnProcessName)) {
			return true
		}
	}
	return false
}

// GetActiveGame detects the currently active game from the GFN window title.
func (d *Detector) GetActiveGame() string {
	if !d.IsGFNRunning() {
		return ""
	}

	title := d.getGFNWindowTitle()
	if title == "" {
		return ""
	}

	if strings.Contains(title, "Application Launch failed") ||
		strings.Contains(title, "Application resource corrupted") {
		log.Printf("⚠️ GFN error detected: %s", title)
		return ""
	}

	return cleanTitle(title)
}

func cleanTitle(title string) string {
	clean := title

	// Simple, reliable approach: find "GeForce NOW" in the title and strip everything from there
	lower := strings.ToLower(clean)
	gfnIdx := strings.Index(lower, "geforce now")
	if gfnIdx > 0 {
		// Take everything before "GeForce NOW"
		clean = clean[:gfnIdx]

		// Strip trailing prepositions and separators: " on ", " - ", " — ", etc.
		clean = strings.TrimRight(clean, " ")
		for _, suffix := range []string{" on", " en", " in", " via", " -", " –", " —"} {
			if strings.HasSuffix(strings.ToLower(clean), suffix) {
				clean = clean[:len(clean)-len(suffix)]
				break
			}
		}

		// Strip parenthetical tags like "- W2 " at the end
		if idx := strings.LastIndex(clean, " - "); idx > 0 {
			// Check if what follows looks like a tag (e.g., "W2", "S1")
			after := strings.TrimSpace(clean[idx+3:])
			if len(after) <= 4 {
				clean = clean[:idx]
			}
		}
	}

	// Strip trademark symbols
	clean = trademarkRe.ReplaceAllString(clean, "")
	clean = strings.TrimSpace(clean)

	// Ignore generic titles
	lowerClean := strings.ToLower(clean)
	if lowerClean == "" || lowerClean == "geforce now" || lowerClean == "games" ||
		lowerClean == "geforce now - games" || lowerClean == "geforcenow" {
		return ""
	}

	return clean
}

// getGFNWindowTitle tries methods in order of reliability on Wayland.
func (d *Detector) getGFNWindowTitle() string {
	// Method 1: Our GNOME Shell extension D-Bus (Wayland native, most reliable)
	if d.extensionReady {
		if title := d.getGFNTitleExtension(); title != "" {
			if d.lastMethod != "extension" {
				log.Println("🔍 Window detection: using GNOME Shell extension D-Bus")
				d.lastMethod = "extension"
			}
			return title
		}
	}

	// Method 2: xprop (XWayland / X11)
	if title := d.getGFNTitleXprop(); title != "" {
		if d.lastMethod != "xprop" {
			log.Println("🔍 Window detection: using xprop")
			d.lastMethod = "xprop"
		}
		return title
	}

	// Method 3: xdotool (X11)
	if title := d.getGFNTitleXdotool(); title != "" {
		if d.lastMethod != "xdotool" {
			log.Println("🔍 Window detection: using xdotool")
			d.lastMethod = "xdotool"
		}
		return title
	}

	// Method 4: Hyprland IPC
	if title := d.getGFNTitleHyprland(); title != "" {
		if d.lastMethod != "hyprland" {
			log.Println("🔍 Window detection: using Hyprland IPC")
			d.lastMethod = "hyprland"
		}
		return title
	}

	// If no method worked and extension wasn't ready, remind user
	if !d.extensionReady && d.lastMethod != "none" {
		log.Println("⚠️ Cannot detect GFN window title. Please log out and back in to activate the GNOME Shell extension, then restart this program.")
		d.lastMethod = "none"
	}

	return ""
}

// ensureExtension installs and enables the GNOME Shell extension.
func (d *Detector) ensureExtension() {
	// Only relevant on GNOME
	if os.Getenv("XDG_CURRENT_DESKTOP") == "" {
		desktop := os.Getenv("DESKTOP_SESSION")
		if !strings.Contains(strings.ToLower(desktop), "gnome") {
			return
		}
	} else if !strings.Contains(strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP")), "gnome") {
		return
	}

	extDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "gnome-shell", "extensions", extensionUUID)

	// Check if already installed with correct version
	installed := false
	if _, err := os.Stat(filepath.Join(extDir, "extension.js")); err == nil {
		installed = true
	}

	if !installed {
		log.Println("📦 Installing GNOME Shell extension for window title detection...")
		if err := os.MkdirAll(extDir, 0755); err != nil {
			log.Printf("❌ Failed to create extension directory: %v", err)
			return
		}

		// Copy embedded files
		for _, name := range []string{"extension.js", "metadata.json"} {
			data, err := extensionFiles.ReadFile(name)
			if err != nil {
				log.Printf("❌ Failed to read embedded %s: %v", name, err)
				return
			}
			if err := os.WriteFile(filepath.Join(extDir, name), data, 0644); err != nil {
				log.Printf("❌ Failed to write %s: %v", name, err)
				return
			}
		}

		// Enable the extension
		cmd := exec.Command("gnome-extensions", "enable", extensionUUID)
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Printf("⚠️ gnome-extensions enable: %s (%v)", string(out), err)
		}

		log.Println("✅ Extension installed. It will activate after you log out and back in (or restart GNOME Shell).")
	} else {
		// Make sure it's enabled
		cmd := exec.Command("gnome-extensions", "enable", extensionUUID)
		_, _ = cmd.CombinedOutput() // best effort
	}

	// Test if the extension's D-Bus service is available right now
	d.extensionReady = d.testExtensionDBus()
	if d.extensionReady {
		log.Println("✅ GNOME Shell extension D-Bus is active")
	} else {
		log.Println("⚠️ GNOME Shell extension installed but not yet active (requires session restart)")
	}
	d.extensionChecked = true
}

// testExtensionDBus checks if our extension's D-Bus service responds.
func (d *Detector) testExtensionDBus() bool {
	cmd := exec.Command("gdbus", "call",
		"--session",
		"--dest", "org.gnome.Shell",
		"--object-path", "/com/geforcenow/WindowTitles",
		"--method", "com.geforcenow.WindowTitles.GetGFNTitle",
	)
	_, err := cmd.Output()
	return err == nil
}

// getGFNTitleExtension calls our GNOME Shell extension via D-Bus.
func (d *Detector) getGFNTitleExtension() string {
	cmd := exec.Command("gdbus", "call",
		"--session",
		"--dest", "org.gnome.Shell",
		"--object-path", "/com/geforcenow/WindowTitles",
		"--method", "com.geforcenow.WindowTitles.GetGFNTitle",
	)
	out, err := cmd.Output()
	if err != nil {
		// Extension might have been unloaded
		if d.extensionReady {
			d.extensionReady = false
			log.Println("⚠️ GNOME Shell extension D-Bus lost, falling back to other methods")
		}
		return ""
	}

	return parseGDBusStringResult(string(out))
}

// getGFNTitleXprop enumerates all X11/XWayland windows.
func (d *Detector) getGFNTitleXprop() string {
	cmd := exec.Command("xprop", "-root", "_NET_CLIENT_LIST")
	cmd.Env = append(os.Environ(), "DISPLAY=:0")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	line := strings.TrimSpace(string(out))
	parts := strings.SplitN(line, "#", 2)
	if len(parts) < 2 {
		return ""
	}

	windowIDs := strings.Split(parts[1], ",")
	for _, wid := range windowIDs {
		wid = strings.TrimSpace(wid)
		if wid == "" {
			continue
		}

		classCmd := exec.Command("xprop", "-id", wid, "WM_CLASS")
		classCmd.Env = append(os.Environ(), "DISPLAY=:0")
		classOut, err := classCmd.Output()
		if err != nil {
			continue
		}

		classStr := strings.ToLower(string(classOut))
		if !strings.Contains(classStr, "geforce") && !strings.Contains(classStr, "geforcenow") {
			continue
		}

		titleCmd := exec.Command("xprop", "-id", wid, "_NET_WM_NAME")
		titleCmd.Env = append(os.Environ(), "DISPLAY=:0")
		titleOut, err := titleCmd.Output()
		if err != nil {
			continue
		}

		title := parseXpropString(string(titleOut))
		if title != "" {
			return title
		}
	}
	return ""
}

// getGFNTitleXdotool uses xdotool (X11/XWayland).
func (d *Detector) getGFNTitleXdotool() string {
	cmd := exec.Command("xdotool", "search", "--class", "geforce")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		wid := strings.TrimSpace(scanner.Text())
		if wid == "" {
			continue
		}
		nameCmd := exec.Command("xdotool", "getwindowname", wid)
		nameOut, err := nameCmd.Output()
		if err != nil {
			continue
		}
		title := strings.TrimSpace(string(nameOut))
		if title != "" {
			return title
		}
	}
	return ""
}

// getGFNTitleHyprland uses Hyprland's IPC.
func (d *Detector) getGFNTitleHyprland() string {
	if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") == "" {
		return ""
	}

	cmd := exec.Command("hyprctl", "clients", "-j")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	outStr := string(out)
	if !strings.Contains(strings.ToLower(outStr), "geforce") {
		return ""
	}

	lines := strings.Split(outStr, "\n")
	inGFN := false
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, `"class":`) && (strings.Contains(lower, "geforce") || strings.Contains(lower, "geforcenow")) {
			inGFN = true
		}
		if inGFN && strings.Contains(lower, `"title"`) {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				title := strings.TrimSpace(parts[1])
				title = strings.Trim(title, `",`)
				return title
			}
		}
	}
	return ""
}

// parseGDBusStringResult parses gdbus output like: ('Some Title',)
func parseGDBusStringResult(output string) string {
	output = strings.TrimSpace(output)
	// Format: ('title',)
	start := strings.Index(output, "'")
	if start < 0 {
		return ""
	}
	end := strings.LastIndex(output, "'")
	if end <= start {
		return ""
	}
	title := output[start+1 : end]
	title = strings.ReplaceAll(title, "\\'", "'")
	return strings.TrimSpace(title)
}

// parseXpropString extracts string from: _NET_WM_NAME(UTF8_STRING) = "title"
func parseXpropString(output string) string {
	idx := strings.Index(output, "= \"")
	if idx < 0 {
		return ""
	}
	rest := output[idx+3:]
	endIdx := strings.LastIndex(rest, "\"")
	if endIdx < 0 {
		return ""
	}
	return rest[:endIdx]
}
