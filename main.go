package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/joshmckinney/geforcenow-presence/internal/config"
	"github.com/joshmckinney/geforcenow-presence/internal/dbussvc"
	"github.com/joshmckinney/geforcenow-presence/internal/detector"
	"github.com/joshmckinney/geforcenow-presence/internal/discord"
	"github.com/joshmckinney/geforcenow-presence/internal/i18n"
	"github.com/joshmckinney/geforcenow-presence/internal/launcher"
	"github.com/joshmckinney/geforcenow-presence/internal/presence"
	"github.com/joshmckinney/geforcenow-presence/internal/ui"
	"github.com/joshmckinney/geforcenow-presence/internal/updater"
)

const version = "0.1.0-beta"

func main() {
	delay := flag.Int("delay", 0, "Delay startup by N seconds")
	interval := flag.Int("interval", 10, "Polling interval in seconds")
	flag.Parse()

	configDir, assetDir := getPaths()

	logDir := filepath.Join(configDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating log directory: %v\n", err)
	}

	logFile, err := os.OpenFile(
		filepath.Join(logDir, "geforce_presence.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime)

	multiWriter := &multiWriter{writers: []writeCloser{
		{w: os.Stdout},
		{w: logFile},
	}}
	log.SetOutput(multiWriter)

	log.Printf("GeForce NOW Rich Presence v%s (Linux/Go)", version)

	lockFile := filepath.Join(os.TempDir(), "geforce_presence.lock")
	if !acquireLock(lockFile) {
		log.Println("⚠️ Another instance is already running. Exiting.")
		os.Exit(0)
	}
	defer releaseLock(lockFile)

	configMgr := config.NewManager(configDir)
	settings := configMgr.GetSettings()
	ui.RebuildIcons(settings.StatusColors)

	if *delay == 0 {
		*delay = settings.StartupDelay
	}
	if *delay > 0 {
		log.Printf("⏳ Waiting %d seconds before starting...", *delay)
		time.Sleep(time.Duration(*delay) * time.Second)
	}

	langDir := filepath.Join(assetDir, "lang")
	lang := i18n.DetectLanguage(settings.Language)
	i18n.LoadLocale(langDir, lang)
	log.Printf("🌐 Language: %s", lang)

	if settings.StartDiscordOnLaunch {
		launcher.LaunchDiscord()
	}
	if settings.StartGFNOnLaunch {
		launcher.LaunchGFN()
	}

	det := detector.New()

	dbusChan, err := dbussvc.StartDBusService()
	if err != nil {
		log.Printf("⚠️ DBus service could not start: %v", err)
	}

	uiActs := ui.StartTray(configMgr, langDir)
	defer ui.QuitTray()

	globalOverrideChan := make(chan string, 10)

	go func() {
		for {
			select {
			case override := <-uiActs.OverrideChan:
				globalOverrideChan <- override
			case override := <-dbusChan:
				globalOverrideChan <- override
			case val := <-uiActs.ToggleConfigGFN:
				s := configMgr.GetSettings()
				s.StartGFNOnLaunch = val
				configMgr.SetSettings(s)
			case val := <-uiActs.ToggleConfigDisc:
				s := configMgr.GetSettings()
				s.StartDiscordOnLaunch = val
				configMgr.SetSettings(s)
			case lang := <-uiActs.ChangeLanguage:
				s := configMgr.GetSettings()
				s.Language = lang
				configMgr.SetSettings(s)
				log.Printf("🌐 Language changed to %s. Restarting...", lang)
				restartApp()
			case val := <-uiActs.SetInterval:
				s := configMgr.GetSettings()
				s.PollingInterval = val
				configMgr.SetSettings(s)
				ui.UpdateIntervalItems(val)
				log.Printf("⏱️ Polling interval set to %ds", val)
			case val := <-uiActs.SetDelay:
				s := configMgr.GetSettings()
				s.StartupDelay = val
				configMgr.SetSettings(s)
				ui.UpdateDelayItems(val)
				log.Printf("⏳ Startup delay set to %ds", val)
			case <-uiActs.OpenConfigDir:
				openConfigDir(configDir)
			case val := <-uiActs.ToggleAutoStart:
				toggleAutoStart(val)
			case <-uiActs.UpdateClicked:
				openURL(updater.GetReleasesURL())
			case colors := <-uiActs.SetColor:
				s := configMgr.GetSettings()
				for k, v := range colors {
					if s.StatusColors == nil {
						s.StatusColors = make(map[string]string)
					}
					s.StatusColors[k] = v
				}
				configMgr.SetSettings(s)
				ui.RebuildIcons(s.StatusColors)
				log.Printf("🎨 Colors updated: %v", colors)
			case <-uiActs.CheckUpdates:
				log.Println("🔍 Manual update check requested")
				newTag, err := updater.CheckForUpdate(version)
				if err != nil {
					ui.ShowMessage("Update Check", "Error checking for updates: "+err.Error())
				} else if newTag != "" {
					ui.ShowUpdateAvailable(newTag)
				} else {
					ui.ShowMessage(i18n.T("tray_title", "GeForce NOW Presence"), i18n.T("update_up_to_date", "You are on the latest version!"))
				}
			}
	}
	}()

	// Background Update check
	go func() {
		// Wait a bit before checking so we don't slow down startup
		time.Sleep(3 * time.Second)
		newTag, err := updater.CheckForUpdate(version)
		if err == nil && newTag != "" {
			ui.ShowUpdateAvailable(newTag)
		}
	}()

	cacheFile := filepath.Join(configDir, "discord_apps_cache.json")
	appsCache := discord.NewAppsCache(cacheFile)

	pInterval := *interval
	if pInterval == 10 && settings.PollingInterval != 0 {
		pInterval = settings.PollingInterval
	}
	pollInterval := time.Duration(pInterval) * time.Second
	pm := presence.New(configMgr, det, appsCache, pollInterval)

	stop := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigChan:
			log.Printf("📛 Received signal, shutting down")
		case <-uiActs.QuitChan:
			log.Printf("📛 User requested exit via Tray")
		}
		close(stop)
	}()

	pm.Run(stop, globalOverrideChan)
	log.Println("👋 GeForce NOW Rich Presence stopped")
}

func getPaths() (string, string) {
	// 1. Determine Config Dir (Always writeable for state/settings)
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		xdgConfig = filepath.Join(os.Getenv("HOME"), ".config")
	}
	userConfigDir := filepath.Join(xdgConfig, "geforcenow-presence")

	// 2. Determine Asset Dir (lang files, shared resources)
	// Priority: 
	// - Local install (next to binary / .local)
	// - System install (/usr/share)

	// Local check (installed via make install or running from source)
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)
	localLang := filepath.Join(exeDir, "lang")
	if _, err := os.Stat(localLang); err == nil {
		return userConfigDir, exeDir
	}

	// Development check (running from repo root)
	wd, _ := os.Getwd()
	if _, err := os.Stat(filepath.Join(wd, "lang")); err == nil {
		return userConfigDir, wd
	}

	// System-wide install (/usr/share)
	systemAssetDir := "/usr/share/geforcenow-presence"
	if _, err := os.Stat(filepath.Join(systemAssetDir, "lang")); err == nil {
		return userConfigDir, systemAssetDir
	}

	return userConfigDir, "."
}

func acquireLock(lockFile string) bool {
	data, err := os.ReadFile(lockFile)
	if err == nil {
		pid, err := strconv.Atoi(string(data))
		if err == nil {
			if proc, err := os.FindProcess(pid); err == nil {
				if err := proc.Signal(syscall.Signal(0)); err == nil {
					return false
				}
			}
		}
		os.Remove(lockFile)
	}
	return os.WriteFile(lockFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644) == nil
}

func releaseLock(lockFile string) {
	os.Remove(lockFile)
}

type writeCloser struct {
	w interface{ Write([]byte) (int, error) }
}

type multiWriter struct {
	writers []writeCloser
}

func (mw *multiWriter) Write(p []byte) (int, error) {
	for _, w := range mw.writers {
		_, _ = w.w.Write(p)
	}
	return len(p), nil
}

func restartApp() {
	cmd := exec.Command("systemctl", "--user", "--no-block", "restart", "geforcenow-presence")
	if err := cmd.Run(); err == nil {
		return
	}

	exe, _ := os.Executable()
	if err := exec.Command(exe).Start(); err != nil {
		log.Printf("❌ Failed to start process: %v", err)
	}
	os.Exit(0)
}

func openConfigDir(path string) {
	openURL(path)
}

func openURL(url string) {
	if err := exec.Command("xdg-open", url).Start(); err != nil {
		log.Printf("❌ Failed to open URL/Path: %v", err)
	}
}

func toggleAutoStart(enable bool) {
	action := "disable"
	if enable {
		action = "enable"
	}
	cmd := exec.Command("systemctl", "--user", action, "geforcenow-presence")
	if err := cmd.Run(); err != nil {
		log.Printf("❌ Failed to %s auto-start: %v", action, err)
	} else {
		log.Printf("✅ Auto-start %sd", action)
	}
}
