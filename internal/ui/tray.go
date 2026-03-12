package ui

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joshmckinney/geforcenow-presence/internal/config"
	"github.com/joshmckinney/geforcenow-presence/internal/i18n"
	"github.com/getlantern/systray"
)

// Actions represents the tray actions sent back to main
type Actions struct {
	OverrideChan     chan string
	ToggleConfigGFN  chan bool
	ToggleConfigDisc chan bool
	ChangeLanguage   chan string
	SetInterval      chan int
	SetDelay         chan int
	OpenConfigDir    chan struct{}
	ToggleAutoStart  chan bool
	QuitChan         chan struct{}
}

var acts Actions
var configMgr *config.Manager
var sysLangDir string
var mPlaying *systray.MenuItem
var mInterval *systray.MenuItem
var mDelay *systray.MenuItem

// StartTray initializes and starts the system tray.
func StartTray(mgr *config.Manager, langDir string) Actions {
	configMgr = mgr
	sysLangDir = langDir
	acts = Actions{
		OverrideChan:     make(chan string, 1),
		ToggleConfigGFN:  make(chan bool, 1),
		ToggleConfigDisc: make(chan bool, 1),
		ChangeLanguage:   make(chan string, 1),
		SetInterval:      make(chan int, 1),
		SetDelay:         make(chan int, 1),
		OpenConfigDir:    make(chan struct{}, 1),
		ToggleAutoStart:  make(chan bool, 1),
		QuitChan:         make(chan struct{}),
	}
	go systray.Run(onReady, onExit)
	return acts
}

// QuitTray exits the system tray.
func QuitTray() {
	systray.Quit()
}

// SetStatus updates the tray icon and tooltip based on the current state.
func SetStatus(state string, gameName string) {
	switch state {
	case "playing":
		systray.SetIcon(iconGreen)
		systray.SetTooltip(i18n.T("tooltip_playing", "GeForce NOW: Playing ") + gameName)
		if mPlaying != nil {
			mPlaying.SetTitle(i18n.T("status_playing", "Playing: ") + gameName)
			mPlaying.Show()
		}
	case "waiting":
		systray.SetIcon(iconYellow)
		systray.SetTooltip(i18n.T("tooltip_waiting", "GeForce NOW: Waiting for game..."))
		if mPlaying != nil {
			mPlaying.SetTitle(i18n.T("status_idle", "Status: Idle"))
			mPlaying.Show()
		}
	case "error":
		systray.SetIcon(iconRed)
		systray.SetTooltip(i18n.T("tooltip_error", "GeForce NOW: Discord RPC Error"))
		if mPlaying != nil {
			mPlaying.SetTitle(i18n.T("status_error", "Status: Discord Error"))
			mPlaying.Show()
		}
	case "disconnected":
		systray.SetIcon(iconRed)
		systray.SetTooltip(i18n.T("tooltip_disconnected", "GeForce NOW: Not Running"))
		if mPlaying != nil {
			mPlaying.SetTitle(i18n.T("status_disconnected", "Status: Not Running"))
			mPlaying.Show()
		}
	}
}

func onReady() {
	systray.SetIcon(iconYellow)
	systray.SetTitle(i18n.T("tray_title", "GeForce NOW Presence"))
	systray.SetTooltip(i18n.T("tray_title", "GeForce NOW Presence"))

	mPlaying = systray.AddMenuItem(i18n.T("status_initializing", "Status: Initializing..."), "")
	mPlaying.Disable()
	// Always keep mPlaying visible now so the user can see what's going on
	mPlaying.Show()

	systray.AddSeparator()
	mForce := systray.AddMenuItem(i18n.T("tray_force_game", "Force Game Name..."), "")
	mClear := systray.AddMenuItem(i18n.T("tray_clear_override", "Clear Override"), "")

	systray.AddSeparator()
	mLogs := systray.AddMenuItem(i18n.T("tray_open_logs", "Open Logs"), "")

	systray.AddSeparator()
	mLanguage := systray.AddMenuItem(i18n.T("tray_language", "Language"), "")
	currLang := configMgr.GetSettings().Language
	if currLang == "" {
		currLang = i18n.DetectLanguage("")
	}
	
	langs := i18n.GetAvailableLanguages(sysLangDir)
	for code, name := range langs {
		item := mLanguage.AddSubMenuItemCheckbox(name, "", currLang == code)
		
		go func(menuItem *systray.MenuItem, langCode string) {
			for range menuItem.ClickedCh {
				if !menuItem.Checked() {
					acts.ChangeLanguage <- langCode
				}
			}
		}(item, code)
	}

	mConfig := systray.AddMenuItem(i18n.T("tray_config", "Configuration"), "")
	mStartGFN := mConfig.AddSubMenuItemCheckbox(i18n.T("config_start_gfn", "Start GeForce NOW on launch"), "", configMgr.GetSettings().StartGFNOnLaunch)
	mStartDisc := mConfig.AddSubMenuItemCheckbox(i18n.T("config_start_discord", "Start Discord on launch"), "", configMgr.GetSettings().StartDiscordOnLaunch)

	s := configMgr.GetSettings()
	mInterval = mConfig.AddSubMenuItem(fmt.Sprintf(i18n.T("tray_polling_interval", "Interval: %ds"), s.PollingInterval), "")
	mDelay = mConfig.AddSubMenuItem(fmt.Sprintf(i18n.T("tray_startup_delay", "Delay: %ds"), s.StartupDelay), "")

	mOpenConfig := mConfig.AddSubMenuItem(i18n.T("tray_open_config_dir", "Open Configuration Folder"), "")

	mAutoStart := mConfig.AddSubMenuItemCheckbox(i18n.T("tray_auto_start", "Auto-start on Login"), "", isAutoStartEnabled())

	systray.AddSeparator()
	mExit := systray.AddMenuItem(i18n.T("tray_exit", "Exit"), "")

	go func() {
		for {
			select {
			case <-mForce.ClickedCh:
				input := promptForString(i18n.T("force_game_prompt", "What game will you force today?"), "")
				if input != "" {
					acts.OverrideChan <- input
				}
			case <-mClear.ClickedCh:
				acts.OverrideChan <- ""
			case <-mLogs.ClickedCh:
				openLogs()
			case <-mStartGFN.ClickedCh:
				val := !mStartGFN.Checked()
				if val {
					mStartGFN.Check()
				} else {
					mStartGFN.Uncheck()
				}
				acts.ToggleConfigGFN <- val
			case <-mStartDisc.ClickedCh:
				val := !mStartDisc.Checked()
				if val {
					mStartDisc.Check()
				} else {
					mStartDisc.Uncheck()
				}
				acts.ToggleConfigDisc <- val
			case <-mInterval.ClickedCh:
				curr := configMgr.GetSettings().PollingInterval
				input := promptForString(i18n.T("tray_custom_prompt_interval", "Enter polling interval in seconds:"), strconv.Itoa(curr))
				if val, err := strconv.Atoi(input); err == nil && val > 0 {
					acts.SetInterval <- val
				}
			case <-mDelay.ClickedCh:
				curr := configMgr.GetSettings().StartupDelay
				input := promptForString(i18n.T("tray_custom_prompt_delay", "Enter startup delay in seconds:"), strconv.Itoa(curr))
				if val, err := strconv.Atoi(input); err == nil && val >= 0 {
					acts.SetDelay <- val
				}
			case <-mOpenConfig.ClickedCh:
				acts.OpenConfigDir <- struct{}{}
			case <-mAutoStart.ClickedCh:
				val := !mAutoStart.Checked()
				if val {
					mAutoStart.Check()
				} else {
					mAutoStart.Uncheck()
				}
				acts.ToggleAutoStart <- val
			case <-mExit.ClickedCh:
				close(acts.QuitChan)
				return
			}
		}
	}()
}

func onExit() {
	// Clean up if needed
}

// UpdateIntervalItems updates the label for the interval menu item.
func UpdateIntervalItems(current int) {
	if mInterval != nil {
		mInterval.SetTitle(fmt.Sprintf(i18n.T("tray_polling_interval", "Interval: %ds"), current))
	}
}

// UpdateDelayItems updates the label for the delay menu item.
func UpdateDelayItems(current int) {
	if mDelay != nil {
		mDelay.SetTitle(fmt.Sprintf(i18n.T("tray_startup_delay", "Delay: %ds"), current))
	}
}

func isAutoStartEnabled() bool {
	cmd := exec.Command("systemctl", "--user", "is-enabled", "geforcenow-presence")
	err := cmd.Run()
	return err == nil
}

func promptForString(prompt string, defaultVal string) string {
	// Try zenity
	args := []string{"--entry", "--text", prompt}
	if defaultVal != "" {
		args = append(args, "--entry-text", defaultVal)
	}
	out, err := exec.Command("zenity", args...).Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	// Try kdialog
	out, err = exec.Command("kdialog", "--inputbox", prompt, defaultVal).Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}

func openLogs() {
	if configMgr == nil {
		return
	}
	logFile := filepath.Join(configMgr.GetConfigDir(), "logs", "geforce_presence.log")
	if err := exec.Command("xdg-open", logFile).Start(); err != nil {
		log.Printf("❌ Failed to open log file: %v", err)
	}
}
