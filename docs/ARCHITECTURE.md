# Architecture & How It Works

## System Overview

```text
┌─────────────────┐     D-Bus      ┌───────────────────────┐
│ GNOME Shell Ext │◄──────────────►│  geforcenow-presence  │
│ (window titles) │   GetGFNTitle  │     (Go binary)       │
└─────────────────┘                 │                       │
                                    │  1. Detect GFN proc   │
┌─────────────────┐     IPC        │  2. Get window title  │
│ Discord Client  │◄──────────────►│  3. Clean game name   │
│ (Rich Presence) │  SET_ACTIVITY  │  4. Lookup metadata   │
└─────────────────┘                 │  5. Update presence   │
                                    └───────────────────────┘
```

## The Pipeline

### 1. Process Detection
A local Linux timer loop scans `/proc` looking for a running `GeForceNOW` background process. The frequency is persistent and configurable (5s - 60s) via the "Polling Interval" menu in the System Tray or the `--interval` flag. These settings are stored in `app_settings.json`.

### 2. Window Title Scraping (Local System D-Bus)
Because Wayland completely isolates applications from reading each other's window titles natively, the application utilizes a tiny generic GNOME Shell extension (`window-title-server@geforcenow-presence`). 

This extension exposes a custom local system D-Bus method allowing the Go monitor to securely query the name of the currently active window. (It also supports fallback mechanisms using `xprop`/`xdotool` on X11 compositors).

### 3. Name Sanitizing
GeForce NOW usually appends `" on GeForce NOW"` along with copyright strings. The application deeply cleans the query using regex matches to yield a clean string (e.g. "Cyberpunk 2077").

### 4. Zero-Auth Metadata Discovery
Instead of bundling a massive static database (or hitting a rate-limited proxy server), this client is entirely **Stateless** and **Zero-Auth**. It connects directly to the public REST endpoints of major game distributors.

**Step A (Steam Search):**
It queries `https://store.steampowered.com/api/storesearch/` with the cleaned title.
If a match is found, it constructs the high-resolution hero image URL:
`https://cdn.akamai.steamstatic.com/steam/apps/<AppID>/header.jpg`

**Step B (GOG Fallback):**
If Steam yields zero results, it cascades to `https://embed.gog.com/games/ajax/filtered`. If matched, it uses GOG's `_glx_master_256.jpg` endpoint.

### 5. Native Discord Presence Hook
Instead of requiring OAuth developer accounts and uploading static PNG assets to Discord's web portal, this client dynamically passes the high-resolution `header.jpg` URL retrieved in Step 4 *directly* into the Discord `large_image` payload.

To fool Discord into natively parsing the game as a recognized application (so it shows up properly in mutual servers and on your user profile), it pulls down a backend cached list of Discord's *Official Desktop App Client IDs* (`detectable_applications.json`).

The Go daemon fuzzy-matches your current window title against the top 20,000 Client IDs. When it finds a match, it instantiates its IPC socket connection *as* that Client ID, tricking Discord into native game integrations with live-updating art.

### 6. Optional Game History (Process Spoofing)
Discord's 30-day "Game History" record requires detecting a local process matching the game's executable name. To support this on Linux/GFN (where no local game process exists):
1. **Dynamic Metadata:** The agent fetches the expected executable name (e.g., `cs2.exe`) directly from the native Discord app record.
2. **Dummy Process:** If `EnableGameHistory` is enabled, the agent launches a light, long-running dummy process (using a system utility like `tail` or `sleep`) renamed to the target executable name in a temporary directory.
3. **PID Handshake:** The PID of this dummy process is passed to the Discord RPC. Discord validates the running process, causing the session to be recorded in the user's activity history.
