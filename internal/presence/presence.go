package presence

import (
	"log"
	"time"

	"github.com/joshmckinney/geforcenow-presence/internal/config"
	"github.com/joshmckinney/geforcenow-presence/internal/detector"
	"github.com/joshmckinney/geforcenow-presence/internal/discord"
	"github.com/joshmckinney/geforcenow-presence/internal/metadata"
	"github.com/joshmckinney/geforcenow-presence/internal/ui"
)

const defaultClientID = "1095416975028650046"

// Manager handles the main presence monitoring loop.
type Manager struct {
	configMgr    *config.Manager
	detector     *detector.Detector
	appsCache    *discord.AppsCache
	rpc          *discord.RPC
	interval     time.Duration
	lastGame     string
	lastImageURL string
	lastClientID string
	overrideGame string
	startTime    int64
	lastLogMsg   string
}

// New creates a new presence manager.
func New(configMgr *config.Manager, det *detector.Detector, appsCache *discord.AppsCache, interval time.Duration) *Manager {
	return &Manager{
		configMgr: configMgr,
		detector:  det,
		appsCache: appsCache,
		interval:  interval,
	}
}

// Run starts the presence monitoring loop. Blocks until stop is signaled.
func (m *Manager) Run(stop <-chan struct{}, overrideChan <-chan string) {
	log.Println("🟢 Starting presence monitor...")

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	m.check()

	for {
		select {
		case override := <-overrideChan:
			m.overrideGame = override
			if override != "" {
				log.Printf("📥 Manual game override activated: %s", override)
			} else {
				log.Println("📥 Manual override cleared. Resuming auto-detection.")
			}
			m.check()
		case <-ticker.C:
			m.check()
		case <-stop:
			log.Println("🛑 Stopping presence monitor...")
			m.cleanup()
			return
		}
	}
}

func (m *Manager) check() {
	var gameName string

	// First check if GFN is running at all
	if !m.detector.IsGFNRunning() {
		if m.lastGame != "" {
			log.Println("🎮 GFN closed, clearing presence")
			m.clearPresence()
			m.lastGame = ""
			m.lastImageURL = ""
			m.lastClientID = ""
			m.startTime = 0
		}
		ui.SetStatus("disconnected", "")
		m.logOnce("⚠️ GeForce NOW is not running")
		return
	}

	// GFN is running, now check for a game
	if m.overrideGame != "" {
		gameName = m.overrideGame
	} else {
		gameName = m.detector.GetActiveGame()
	}

	if gameName == "" {
		if m.lastGame != "" {
			log.Println("🎮 Game ended, clearing presence")
			m.clearPresence()
			m.lastGame = ""
			m.lastImageURL = ""
			m.lastClientID = ""
			m.startTime = 0
		}

		ui.SetStatus("waiting", "")
		m.logOnce("⏳ Waiting for GeForce NOW game...")
		return
	}

	gameChanged := gameName != m.lastGame

	if gameChanged {
		log.Printf("🎮 Game detected: %s", gameName)
		m.startTime = time.Now().Unix()
		m.lastGame = gameName
		m.lastImageURL = metadata.FetchArt(gameName)

		newClientID := defaultClientID
		match := m.appsCache.FindMatchAutoApply(gameName)
		if match != nil {
			newClientID = match.ID
			log.Printf("🔁 Found native Discord match: %s (client_id: %s)", match.Name, match.ID)
		}

		if m.lastClientID != newClientID && m.rpc != nil {
			m.rpc.Close()
			m.rpc = nil
		}
		m.lastClientID = newClientID
	}

	var justConnected bool
	// Connect RPC if needed
	if m.rpc == nil || !m.rpc.IsConnected() {
		clientID := m.lastClientID
		if clientID == "" {
			clientID = defaultClientID
		}
		m.rpc = discord.NewRPC(clientID)
		if err := m.rpc.Connect(); err != nil {
			log.Printf("❌ Discord RPC connection failed: %v", err)
			m.rpc = nil
			ui.SetStatus("error", "")
			return
		}
		log.Printf("🔁 Connected to Discord RPC with client_id=%s", clientID)
		justConnected = true
	}

	// Always update activity if game changed or we just connected
	if gameChanged || justConnected {
		details := gameName
		if m.lastClientID != defaultClientID {
			// If we are using the native client ID, Discord already displays the game name
			// as the top-level header. We don't want to duplicate it.
			details = ""
		}

		activity := &discord.Activity{
			Details:    details,
			State:      "Playing on GeForce NOW",
			LargeImage: m.lastImageURL,
			LargeText:  gameName,
			StartTime:  m.startTime,
		}

		if err := m.rpc.SetActivity(activity); err != nil {
			log.Printf("❌ Error updating presence: %v", err)
			log.Println("🔄 Discord connection lost or socket error, will reconnect...")
			m.rpc.Close()
			m.rpc = nil
		} else {
			log.Printf("✅ Discord Presence updated for: %s", gameName)
			ui.SetStatus("playing", gameName)
		}
	}
}

func (m *Manager) clearPresence() {
	if m.rpc != nil {
		if err := m.rpc.ClearActivity(); err != nil {
			log.Printf("⚠️ Error clearing presence: %v", err)
		}
	}
}

func (m *Manager) cleanup() {
	if m.rpc != nil {
		m.clearPresence()
		m.rpc.Close()
		m.rpc = nil
	}
}

func (m *Manager) logOnce(msg string) {
	if msg != m.lastLogMsg {
		log.Println(msg)
		m.lastLogMsg = msg
	}
}
