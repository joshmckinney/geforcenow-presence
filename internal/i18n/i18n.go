package i18n

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Texts holds the loaded locale strings.
var Texts map[string]string

// LoadLocale loads the locale file for the given language.
// langDir is the path to the lang/ directory.
func LoadLocale(langDir, lang string) {
	Texts = make(map[string]string)

	path := filepath.Join(langDir, lang+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		// Fallback to English
		path = filepath.Join(langDir, "en.json")
		data, err = os.ReadFile(path)
		if err != nil {
			log.Printf("⚠️ Could not load any locale file from %s", langDir)
			return
		}
	}

	if err := json.Unmarshal(data, &Texts); err != nil {
		log.Printf("⚠️ Error parsing locale file %s: %v", path, err)
	}
}

// T returns the translated string for the given key, or the fallback if not found.
func T(key, fallback string) string {
	if v, ok := Texts[key]; ok {
		return v
	}
	return fallback
}

// DetectLanguage returns a language code from config or the LANG environment variable.
func DetectLanguage(configLang string) string {
	if configLang != "" {
		return configLang
	}

	lang := os.Getenv("LANG")
	if lang == "" {
		lang = os.Getenv("GEFORCE_LANG")
	}
	if lang == "" {
		return "en"
	}

	lang = strings.ToLower(lang)
	if strings.Contains(lang, "es") {
		return "es"
	}
	return "en"
}

// GetAvailableLanguages parses the lang directory and returns a map of file basename -> _language_name (or basename if missing)
func GetAvailableLanguages(langDir string) map[string]string {
	langs := make(map[string]string)

	files, err := os.ReadDir(langDir)
	if err != nil {
		return langs
	}

	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
			code := strings.TrimSuffix(f.Name(), ".json")
			path := filepath.Join(langDir, f.Name())

			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			var tmp map[string]string
			if err := json.Unmarshal(data, &tmp); err == nil {
				if name, ok := tmp["_language_name"]; ok && name != "" {
					langs[code] = name
				} else {
					langs[code] = code
				}
			}
		}
	}

	return langs
}
