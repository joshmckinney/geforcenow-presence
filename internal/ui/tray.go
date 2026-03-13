package ui

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/getlantern/systray"
	"github.com/joshmckinney/geforcenow-presence/internal/config"
	"github.com/joshmckinney/geforcenow-presence/internal/i18n"
)

// Actions represents the tray actions sent back to main
type Actions struct {
	OverrideChan      chan string
	ToggleConfigGFN   chan bool
	ToggleConfigDisc  chan bool
	ChangeLanguage    chan string
	SetInterval       chan int
	SetDelay          chan int
	OpenConfigDir     chan struct{}
	ToggleAutoStart   chan bool
	ToggleGameHistory chan bool
	UpdateClicked     chan struct{}
	CheckUpdates      chan struct{}
	SetColor          chan map[string]string
	QuitChan          chan struct{}
}

var acts Actions
var configMgr *config.Manager
var sysLangDir string
var mPlaying *systray.MenuItem
var mInterval *systray.MenuItem
var mDelay *systray.MenuItem
var mUpdate *systray.MenuItem
var appVersion string

// StartTray initializes and starts the system tray.
func StartTray(mgr *config.Manager, langDir, version string) Actions {
	configMgr = mgr
	sysLangDir = langDir
	appVersion = version
	acts = Actions{
		OverrideChan:      make(chan string, 1),
		ToggleConfigGFN:   make(chan bool, 1),
		ToggleConfigDisc:  make(chan bool, 1),
		ChangeLanguage:    make(chan string, 1),
		SetInterval:       make(chan int, 1),
		SetDelay:          make(chan int, 1),
		OpenConfigDir:     make(chan struct{}, 1),
		ToggleAutoStart:   make(chan bool, 1),
		ToggleGameHistory: make(chan bool, 1),
		UpdateClicked:     make(chan struct{}, 1),
		CheckUpdates:      make(chan struct{}, 1),
		SetColor:          make(chan map[string]string, 1),
		QuitChan:          make(chan struct{}),
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

	mUpdate = systray.AddMenuItem(i18n.T("tray_update_available", "Update Available!"), "")
	mUpdate.Hide()

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

	type langItem struct {
		code string
		name string
	}
	var sortedLangs []langItem
	for code, name := range langs {
		sortedLangs = append(sortedLangs, langItem{code, name})
	}
	sort.Slice(sortedLangs, func(i, j int) bool {
		return sortedLangs[i].name < sortedLangs[j].name
	})

	for _, l := range sortedLangs {
		item := mLanguage.AddSubMenuItemCheckbox(l.name, "", currLang == l.code)

		go func(menuItem *systray.MenuItem, langCode string) {
			for range menuItem.ClickedCh {
				if !menuItem.Checked() {
					acts.ChangeLanguage <- langCode
				}
			}
		}(item, l.code)
	}

	mConfig := systray.AddMenuItem(i18n.T("tray_config", "Configuration"), "")
	mStartGFN := mConfig.AddSubMenuItemCheckbox(i18n.T("config_start_gfn", "Start GeForce NOW on launch"), "", configMgr.GetSettings().StartGFNOnLaunch)
	mStartDisc := mConfig.AddSubMenuItemCheckbox(i18n.T("config_start_discord", "Start Discord on launch"), "", configMgr.GetSettings().StartDiscordOnLaunch)

	s := configMgr.GetSettings()
	mInterval = mConfig.AddSubMenuItem(fmt.Sprintf(i18n.T("tray_polling_interval", "Interval: %ds"), s.PollingInterval), "")
	mDelay = mConfig.AddSubMenuItem(fmt.Sprintf(i18n.T("tray_startup_delay", "Delay: %ds"), s.StartupDelay), "")

	mOpenConfig := mConfig.AddSubMenuItem(i18n.T("tray_open_config_dir", "Open Configuration Folder"), "")

	mColors := mConfig.AddSubMenuItem(i18n.T("tray_custom_colors", "Custom Status Colors"), "")
	mPlayCol := mColors.AddSubMenuItem(i18n.T("tray_color_playing", "Playing Color..."), "")
	mWaitCol := mColors.AddSubMenuItem(i18n.T("tray_color_waiting", "Idle Color..."), "")
	mErrCol := mColors.AddSubMenuItem(i18n.T("tray_color_error", "Error Color..."), "")

	mAutoStart := mConfig.AddSubMenuItemCheckbox(i18n.T("tray_auto_start", "Auto-start on Login"), "", isAutoStartEnabled())
	mHistory := mConfig.AddSubMenuItemCheckbox(i18n.T("config_enable_history", "Enable Game History (30-day log)"), "", configMgr.GetSettings().EnableGameHistory)

	mCheck := systray.AddMenuItem(i18n.T("tray_check_updates", "Check for Updates"), "")
	systray.AddSeparator()
	mVersion := systray.AddMenuItem(fmt.Sprintf(i18n.T("tray_version", "Version: %s"), appVersion), "")
	mVersion.Disable()
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
			case <-mHistory.ClickedCh:
				val := !mHistory.Checked()
				if val {
					mHistory.Check()
				} else {
					mHistory.Uncheck()
				}
				acts.ToggleGameHistory <- val
			case <-mUpdate.ClickedCh:
				acts.UpdateClicked <- struct{}{}
			case <-mPlayCol.ClickedCh:
				curr := configMgr.GetSettings().StatusColors["playing"]
				if col := pickColor(curr); col != "" {
					acts.SetColor <- map[string]string{"playing": col}
				}
			case <-mWaitCol.ClickedCh:
				curr := configMgr.GetSettings().StatusColors["waiting"]
				if col := pickColor(curr); col != "" {
					acts.SetColor <- map[string]string{"waiting": col}
				}
			case <-mErrCol.ClickedCh:
				curr := configMgr.GetSettings().StatusColors["error"]
				if col := pickColor(curr); col != "" {
					acts.SetColor <- map[string]string{"error": col}
					acts.SetColor <- map[string]string{"disconnected": col}
				}
			case <-mCheck.ClickedCh:
				acts.CheckUpdates <- struct{}{}
			case <-mExit.ClickedCh:
				close(acts.QuitChan)
				return
			}
		}
	}()
}

// ShowUpdateAvailable makes the update menu item visible with a specific tag name.
func ShowUpdateAvailable(tagName string) {
	if mUpdate != nil {
		mUpdate.SetTitle(fmt.Sprintf(i18n.T("tray_update_to", "Update Available: %s"), tagName))
		mUpdate.Show()
	}
}

// ShowMessage shows a desktop notification/dialog with a message.
func ShowMessage(title, msg string) {
	// Use Zenity for simple info dialog
	_ = exec.Command("zenity", "--info", "--title", title, "--text", msg, "--no-wrap").Run()
}

func pickColor(current string) string {
	out, err := exec.Command("zenity", "--color-selection", "--color", current).Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
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
	logFile := filepath.Join(configMgr.GetStateDir(), "geforce_presence.log")
	if err := exec.Command("xdg-open", logFile).Start(); err != nil {
		log.Printf("❌ Failed to open log file: %v", err)
	}
}
