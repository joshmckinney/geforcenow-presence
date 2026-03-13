package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// AppSettings represents the application settings.
type AppSettings struct {
	Language             string            `json:"language"`
	StartGFNOnLaunch     bool              `json:"start_gfn_on_launch"`
	StartDiscordOnLaunch bool              `json:"start_discord_on_launch"`
	PollingInterval      int               `json:"polling_interval"`
	StartupDelay         int               `json:"startup_delay"`
	EnableGameHistory    bool              `json:"enable_game_history"`
	StatusColors         map[string]string `json:"status_colors"`
}

// Manager handles loading and saving configuration.
type Manager struct {
	mu              sync.RWMutex
	configDir       string
	stateDir        string
	appSettings     AppSettings
	appSettingsPath string
}

// NewManager creates a new config manager.
func NewManager(configDir string, stateDir string) *Manager {
	m := &Manager{
		configDir:       configDir,
		stateDir:        stateDir,
		appSettingsPath: filepath.Join(configDir, "app_settings.json"),
		appSettings: AppSettings{
			Language:             "",
			StartGFNOnLaunch:     false,
			StartDiscordOnLaunch: false,
			PollingInterval:      10,
			StartupDelay:         5,
			EnableGameHistory:    false,
			StatusColors: map[string]string{
				"playing":      "#2ecc71",
				"waiting":      "#f1c40f",
				"error":        "#e74c3c",
				"disconnected": "#e74c3c",
			},
		},
	}
	m.load()
	return m
}

func (m *Manager) load() {
	// Try user config first
	data, err := os.ReadFile(m.appSettingsPath)
	if err == nil {
		if err := json.Unmarshal(data, &m.appSettings); err == nil {
			return
		}
		log.Printf("⚠️ Error parsing user app_settings.json: %v", err)
	}

	// Try system fallback (/etc/geforcenow-presence/app_settings.json)
	systemPath := "/etc/geforcenow-presence/app_settings.json"
	data, err = os.ReadFile(systemPath)
	if err == nil {
		if err := json.Unmarshal(data, &m.appSettings); err == nil {
			log.Println("✅ Loaded system-wide default settings")
			// Immediately save to user dir for persistence
			m.saveAppSettings()
			return
		}
	}

	// Fallback to defaults
	m.saveAppSettings()
}

func (m *Manager) saveAppSettings() {
	data, err := json.MarshalIndent(m.appSettings, "", "    ")
	if err != nil {
		log.Printf("❌ Error marshaling app settings: %v", err)
		return
	}
	if err := os.WriteFile(m.appSettingsPath, data, 0644); err != nil {
		log.Printf("❌ Error saving app_settings.json: %v", err)
	}
}

// GetConfigDir returns the base configuration directory path.
func (m *Manager) GetConfigDir() string {
	return m.configDir
}

// GetStateDir returns the base state directory path.
func (m *Manager) GetStateDir() string {
	return m.stateDir
}

// GetSettings returns a copy of the current app settings.
func (m *Manager) GetSettings() AppSettings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.appSettings
}

// SetSettings updates and saves the app settings.
func (m *Manager) SetSettings(settings AppSettings) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.appSettings = settings
	m.saveAppSettings()
}
