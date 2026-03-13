package discord

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	opHandshake = 0
	opFrame     = 1
	opClose     = 2
	opPing      = 3
	opPong      = 4
)

// RPC is a Discord Rich Presence IPC client.
type RPC struct {
	mu       sync.Mutex
	conn     net.Conn
	clientID string
	nonce    int
}

// NewRPC creates a new Discord RPC client.
func NewRPC(clientID string) *RPC {
	return &RPC{clientID: clientID}
}

// Connect establishes a connection to the Discord IPC socket.
func (r *RPC) Connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn != nil {
		return nil
	}

	socketPath := findDiscordSocket()
	if socketPath == "" {
		return fmt.Errorf("discord IPC socket not found")
	}

	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to Discord IPC: %w", err)
	}
	r.conn = conn

	// Send handshake
	handshake := map[string]string{
		"v":         "1",
		"client_id": r.clientID,
	}
	if err := r.send(opHandshake, handshake); err != nil {
		r.conn.Close()
		r.conn = nil
		return fmt.Errorf("handshake failed: %w", err)
	}

	// Read handshake response
	if _, _, err := r.recv(); err != nil {
		r.conn.Close()
		r.conn = nil
		return fmt.Errorf("handshake response failed: %w", err)
	}

	log.Printf("✅ Connected to Discord RPC (client_id=%s)", r.clientID)
	return nil
}

// SetActivity updates the Discord Rich Presence activity.
func (r *RPC) SetActivity(pid int, activity *Activity) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn == nil {
		return fmt.Errorf("not connected to Discord")
	}

	r.nonce++
	payload := map[string]interface{}{
		"cmd":   "SET_ACTIVITY",
		"nonce": fmt.Sprintf("%d", r.nonce),
		"args": map[string]interface{}{
			"pid":      pid,
			"activity": activity.toMap(),
		},
	}

	if err := r.send(opFrame, payload); err != nil {
		return err
	}

	// Read response (but don't block forever)
	_ = r.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err := r.recv()
	_ = r.conn.SetReadDeadline(time.Time{})
	return err
}

// ClearActivity clears the Discord Rich Presence.
func (r *RPC) ClearActivity(pid int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn == nil {
		return nil
	}

	r.nonce++
	payload := map[string]interface{}{
		"cmd":   "SET_ACTIVITY",
		"nonce": fmt.Sprintf("%d", r.nonce),
		"args": map[string]interface{}{
			"pid": pid,
		},
	}

	if err := r.send(opFrame, payload); err != nil {
		return err
	}

	_ = r.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err := r.recv()
	_ = r.conn.SetReadDeadline(time.Time{})
	return err
}

// Close closes the Discord IPC connection.
func (r *RPC) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn != nil {
		r.conn.Close()
		r.conn = nil
		log.Println("🔴 Discord RPC connection closed")
	}
}

// IsConnected returns true if the RPC client is connected.
func (r *RPC) IsConnected() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.conn != nil
}

// Reconnect reconnects with a potentially different client ID.
func (r *RPC) Reconnect(clientID string) error {
	r.Close()
	r.mu.Lock()
	r.clientID = clientID
	r.mu.Unlock()
	return r.Connect()
}

func (r *RPC) send(opcode int, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header[0:4], uint32(opcode))
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(payload)))

	buf := bytes.NewBuffer(header)
	buf.Write(payload)

	_, err = r.conn.Write(buf.Bytes())
	return err
}

func (r *RPC) recv() (int, []byte, error) {
	header := make([]byte, 8)
	if _, err := r.conn.Read(header); err != nil {
		return 0, nil, err
	}

	opcode := int(binary.LittleEndian.Uint32(header[0:4]))
	length := int(binary.LittleEndian.Uint32(header[4:8]))

	if length > 1024*1024 {
		return 0, nil, fmt.Errorf("response too large: %d bytes", length)
	}

	data := make([]byte, length)
	totalRead := 0
	for totalRead < length {
		n, err := r.conn.Read(data[totalRead:])
		if err != nil {
			return 0, nil, err
		}
		totalRead += n
	}

	return opcode, data, nil
}

func findDiscordSocket() string {
	// Check XDG_RUNTIME_DIR first (standard location)
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}

	// Try discord-ipc-0 through discord-ipc-9
	for i := 0; i < 10; i++ {
		path := filepath.Join(runtimeDir, fmt.Sprintf("discord-ipc-%d", i))
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Also check /tmp as fallback
	for i := 0; i < 10; i++ {
		path := filepath.Join("/tmp", fmt.Sprintf("discord-ipc-%d", i))
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Check snap paths
	snapDir := filepath.Join(runtimeDir, "snap.discord")
	if info, err := os.Stat(snapDir); err == nil && info.IsDir() {
		for i := 0; i < 10; i++ {
			path := filepath.Join(snapDir, fmt.Sprintf("discord-ipc-%d", i))
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	return ""
}

// Activity represents a Discord Rich Presence activity.
type Activity struct {
	Details    string
	State      string
	LargeImage string
	LargeText  string
	SmallImage string
	StartTime  int64
	PartySize  []int // [current, max]
}

func (a *Activity) toMap() map[string]interface{} {
	m := make(map[string]interface{})

	if a.Details != "" {
		m["details"] = a.Details
	}
	if a.State != "" {
		m["state"] = a.State
	}
	if a.StartTime > 0 {
		m["timestamps"] = map[string]int64{
			"start": a.StartTime,
		}
	}

	assets := make(map[string]string)
	if a.LargeImage != "" {
		assets["large_image"] = a.LargeImage
	}
	if a.LargeText != "" {
		assets["large_text"] = a.LargeText
	}
	if a.SmallImage != "" {
		assets["small_image"] = a.SmallImage
	}
	if len(assets) > 0 {
		m["assets"] = assets
	}

	if len(a.PartySize) == 2 && a.PartySize[1] > 0 {
		m["party"] = map[string]interface{}{
			"id":   "gfn",
			"size": a.PartySize,
		}
	}

	return m
}
