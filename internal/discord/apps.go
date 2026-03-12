package discord

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	discordDetectableURL = "https://discord.com/api/v9/applications/detectable"
	cacheTTL             = 24 * time.Hour
	autoApplyThreshold   = 0.55
)

// DiscordApp represents a detectable Discord application.
type DiscordApp struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases"`
	Executables []struct {
		Name string `json:"name"`
		OS   string `json:"os"`
	} `json:"executables"`
}

// MatchResult represents a fuzzy match result.
type MatchResult struct {
	Name  string
	ID    string
	Exe   string
	Score float64
}

// AppsCache manages the Discord detectable apps cache.
type AppsCache struct {
	mu        sync.RWMutex
	apps      []DiscordApp
	cacheFile string
	lastFetch time.Time
	client    *http.Client
}

type cacheData struct {
	Timestamp int64        `json:"_ts"`
	Apps      []DiscordApp `json:"apps"`
}

// NewAppsCache creates a new apps cache manager.
func NewAppsCache(cacheFile string) *AppsCache {
	return &AppsCache{
		cacheFile: cacheFile,
		client:    &http.Client{Timeout: 15 * time.Second},
	}
}

// GetApps returns the cached apps, fetching if needed.
func (c *AppsCache) GetApps(forceDownload bool) []DiscordApp {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !forceDownload && len(c.apps) > 0 && time.Since(c.lastFetch) < cacheTTL {
		return c.apps
	}

	// Try loading from file cache
	if !forceDownload {
		if apps := c.loadFromFile(); len(apps) > 0 {
			c.apps = apps
			return c.apps
		}
	}

	// Fetch from API
	apps, err := c.fetchFromAPI()
	if err != nil {
		log.Printf("⚠️ Error fetching Discord apps: %v", err)
		return c.apps
	}

	c.apps = apps
	c.lastFetch = time.Now()
	c.saveToFile()

	log.Printf("✅ Discord apps cache updated (%d apps)", len(apps))
	return c.apps
}

func (c *AppsCache) loadFromFile() []DiscordApp {
	data, err := os.ReadFile(c.cacheFile)
	if err != nil {
		return nil
	}

	var cache cacheData
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil
	}

	if time.Since(time.Unix(cache.Timestamp, 0)) > cacheTTL {
		return nil
	}

	c.lastFetch = time.Unix(cache.Timestamp, 0)
	return cache.Apps
}

func (c *AppsCache) saveToFile() {
	cache := cacheData{
		Timestamp: time.Now().Unix(),
		Apps:      c.apps,
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return
	}
	if err := os.WriteFile(c.cacheFile, data, 0644); err != nil {
		log.Printf("⚠️ Error writing apps cache: %v", err)
	}
}

func (c *AppsCache) fetchFromAPI() ([]DiscordApp, error) {
	log.Println("⬇️ Downloading Discord detectable apps...")

	req, err := http.NewRequest("GET", discordDetectableURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "GeForcePresence/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("discord API returned status %d", resp.StatusCode)
	}

	var apps []DiscordApp
	if err := json.NewDecoder(resp.Body).Decode(&apps); err != nil {
		return nil, err
	}

	return apps, nil
}

// FindMatch finds the best matching Discord app for a game name.
// Returns the best match if it meets the auto-apply threshold.
func (c *AppsCache) FindMatch(gameName string) *MatchResult {
	apps := c.GetApps(false)
	if len(apps) == 0 {
		return nil
	}

	gnl := strings.ToLower(gameName)
	var best *MatchResult

	for _, app := range apps {
		nameL := strings.ToLower(app.Name)

		// Calculate similarity score
		score := similarity(gnl, nameL)

		// Check aliases
		for _, alias := range app.Aliases {
			if s := similarity(gnl, strings.ToLower(alias)); s > score {
				score = s
			}
		}

		if score < 0.35 {
			continue
		}

		if best == nil || score > best.Score {
			exe := ""
			// Look for linux executable first, then win32 as fallback
			for _, e := range app.Executables {
				if e.OS == "linux" && e.Name != "" {
					exe = e.Name
					break
				}
			}
			if exe == "" {
				for _, e := range app.Executables {
					if e.OS == "win32" && e.Name != "" {
						exe = e.Name
						break
					}
				}
			}

			best = &MatchResult{
				Name:  app.Name,
				ID:    app.ID,
				Exe:   exe,
				Score: score,
			}
		}
	}

	if best != nil && best.Score >= autoApplyThreshold {
		return best
	}

	return best
}

// FindMatchAutoApply returns a match only if it meets the auto-apply threshold.
func (c *AppsCache) FindMatchAutoApply(gameName string) *MatchResult {
	match := c.FindMatch(gameName)
	if match != nil && match.Score >= autoApplyThreshold {
		return match
	}
	return nil
}

// similarity calculates a simple similarity ratio between two strings.
// This is a basic implementation similar to difflib.SequenceMatcher.
func similarity(a, b string) float64 {
	if a == b {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	// Check for exact substring match
	if strings.Contains(a, b) || strings.Contains(b, a) {
		shorter := len(a)
		if len(b) < shorter {
			shorter = len(b)
		}
		longer := len(a)
		if len(b) > longer {
			longer = len(b)
		}
		return float64(shorter) / float64(longer)
	}

	// Levenshtein-based similarity
	d := levenshtein(a, b)
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	return 1.0 - float64(d)/float64(maxLen)
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Use two rows for space efficiency
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
