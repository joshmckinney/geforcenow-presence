package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNewManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	if m == nil {
		t.Fatal("Expected NewManager to return a non-nil manager")
	}

	if m.GetConfigDir() != tmpDir {
		t.Errorf("Expected config dir %s, got %s", tmpDir, m.GetConfigDir())
	}

	settings := m.GetSettings()
	if settings.PollingInterval != 10 {
		t.Errorf("Expected default PollingInterval 10, got %d", settings.PollingInterval)
	}
}

func TestSaveAndLoadSettings(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test_save")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	newSettings := AppSettings{
		Language:             "fr",
		StartGFNOnLaunch:     true,
		StartDiscordOnLaunch: true,
		PollingInterval:      30,
		StartupDelay:         10,
	}

	m.SetSettings(newSettings)

	// Create a new manager to verify loading from disk
	m2 := NewManager(tmpDir)
	loadedSettings := m2.GetSettings()

	if !reflect.DeepEqual(loadedSettings, newSettings) {
		t.Errorf("Expected %+v, got %+v", newSettings, loadedSettings)
	}
}

func TestLoadExistingFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test_existing")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	settingsPath := filepath.Join(tmpDir, "app_settings.json")
	content := `{
		"language": "de",
		"start_gfn_on_launch": true,
		"polling_interval": 60
	}`
	if err := os.WriteFile(settingsPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write mock config file: %v", err)
	}

	m := NewManager(tmpDir)
	settings := m.GetSettings()

	if settings.Language != "de" {
		t.Errorf("Expected language 'de', got '%s'", settings.Language)
	}
	if settings.PollingInterval != 60 {
		t.Errorf("Expected PollingInterval 60, got %d", settings.PollingInterval)
	}
	if settings.StartGFNOnLaunch != true {
		t.Error("Expected StartGFNOnLaunch true")
	}
}
