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
	Language             string `json:"language"`
	StartGFNOnLaunch     bool   `json:"start_gfn_on_launch"`
	StartDiscordOnLaunch bool   `json:"start_discord_on_launch"`
	PollingInterval      int    `json:"polling_interval"`
	StartupDelay         int    `json:"startup_delay"`
}

// Manager handles loading and saving configuration.
type Manager struct {
	mu              sync.RWMutex
	configDir       string
	appSettings     AppSettings
	appSettingsPath string
}

// NewManager creates a new config manager.
func NewManager(configDir string) *Manager {
	m := &Manager{
		configDir:       configDir,
		appSettingsPath: filepath.Join(configDir, "app_settings.json"),
		appSettings: AppSettings{
			Language:             "",
			StartGFNOnLaunch:     false,
			StartDiscordOnLaunch: false,
			PollingInterval:      10,
			StartupDelay:         5,
		},
	}
	m.load()
	return m
}

func (m *Manager) load() {
	data, err := os.ReadFile(m.appSettingsPath)
	if err == nil {
		if err := json.Unmarshal(data, &m.appSettings); err != nil {
			log.Printf("⚠️ Error parsing app_settings.json: %v", err)
		}
	} else {
		m.saveAppSettings()
	}
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
