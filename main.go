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
)

const version = "1.0.0"

func main() {
	delay := flag.Int("delay", 0, "Delay startup by N seconds")
	interval := flag.Int("interval", 10, "Polling interval in seconds")
	flag.Parse()

	baseDir := getBaseDir()

	logDir := filepath.Join(baseDir, "logs")
	os.MkdirAll(logDir, 0755)

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

	configDir := filepath.Join(baseDir, "config")
	if _, err := os.Stat(filepath.Join(configDir, "app_settings.json")); err != nil {
		configDir = baseDir
	}
	configMgr := config.NewManager(configDir)
	settings := configMgr.GetSettings()

	if *delay == 0 {
		*delay = settings.StartupDelay
	}
	if *delay > 0 {
		log.Printf("⏳ Waiting %d seconds before starting...", *delay)
		time.Sleep(time.Duration(*delay) * time.Second)
	}

	langDir := filepath.Join(baseDir, "lang")
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
			}
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

func getBaseDir() string {
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		xdgConfig = filepath.Join(os.Getenv("HOME"), ".config")
	}
	installedDir := filepath.Join(xdgConfig, "geforcenow-presence")
	if _, err := os.Stat(filepath.Join(installedDir, "app_settings.json")); err == nil {
		return installedDir
	}

	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(exeDir, "config")); err == nil {
			return exeDir
		}
	}

	wd, err := os.Getwd()
	if err == nil {
		if _, err := os.Stat(filepath.Join(wd, "config")); err == nil {
			return wd
		}
	}

	return "."
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
		w.w.Write(p)
	}
	return len(p), nil
}

func restartApp() {
	cmd := exec.Command("systemctl", "--user", "--no-block", "restart", "geforcenow-presence")
	if err := cmd.Run(); err == nil {
		return
	}

	exe, _ := os.Executable()
	exec.Command(exe).Start()
	os.Exit(0)
}

func openConfigDir(path string) {
	exec.Command("xdg-open", path).Start()
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
