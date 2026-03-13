package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name       string
		configLang string
		envLang    string
		expected   string
	}{
		{"Config overrides all", "fr", "en_US.UTF-8", "fr"},
		{"ENV es", "", "es_ES.UTF-8", "es"},
		{"ENV case insensitive", "", "ES", "es"},
		{"ENV default en", "", "ru_RU.UTF-8", "en"},
		{"Empty fallback", "", "", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("LANG", tt.envLang)
			got := DetectLanguage(tt.configLang)
			if got != tt.expected {
				t.Errorf("DetectLanguage() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestT(t *testing.T) {
	Texts = map[string]string{
		"hello": "bonjour",
	}

	if T("hello", "hi") != "bonjour" {
		t.Error("Expected translation for 'hello'")
	}
	if T("missing", "fallback") != "fallback" {
		t.Error("Expected fallback for missing key")
	}
}

func TestLoadLocale(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lang_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	enData := `{"hello": "hello", "_language_name": "English"}`
	esData := `{"hello": "hola", "_language_name": "Español"}`

	if err := os.WriteFile(filepath.Join(tmpDir, "en.json"), []byte(enData), 0644); err != nil {
		t.Fatalf("Failed to write mock config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "es.json"), []byte(esData), 0644); err != nil {
		t.Fatalf("Failed to write mock config: %v", err)
	}

	LoadLocale(tmpDir, "es")
	if T("hello", "") != "hola" {
		t.Errorf("Expected 'hola', got '%s'", T("hello", ""))
	}

	LoadLocale(tmpDir, "nonexistent")
	if T("hello", "") != "hello" {
		t.Errorf("Expected fallback to 'hello', got '%s'", T("hello", ""))
	}
}

func TestGetAvailableLanguages(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lang_avail_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	enData := `{"_language_name": "English"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "en.json"), []byte(enData), 0644); err != nil {
		t.Fatalf("Failed to write mock config: %v", err)
	}

	langNoName := `{"key": "value"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "de.json"), []byte(langNoName), 0644); err != nil {
		t.Fatalf("Failed to write mock config: %v", err)
	}

	langs := GetAvailableLanguages(tmpDir)
	if langs["en"] != "English" {
		t.Errorf("Expected English, got %s", langs["en"])
	}
	if langs["de"] != "de" {
		t.Errorf("Expected de, got %s", langs["de"])
	}
}
