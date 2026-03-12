# Extending & Modifying

## Project Structure

```text
geforcenow-presence/
├── main.go                          # Entry point, CLI flags, logging, signals
├── Makefile                         # Build, install, uninstall targets
├── go.mod                           # Go module (zero external dependencies)
├── internal/
│   ├── config/config.go             # Persistent settings (Interval, Delay, Auto-start)
│   ├── detector/detector.go         # Window title detection (multi-method)
│   ├── detector/extension.js        # GNOME Shell extension (embedded in binary)
│   ├── detector/metadata.json       # Extension metadata (embedded in binary)
│   ├── discord/rpc.go               # Discord IPC client (Unix socket)
│   ├── discord/apps.go              # Discord detectable apps cache + fuzzy match
│   ├── i18n/i18n.go                 # Locale detection and translation loading
│   ├── launcher/launcher.go         # GFN Flatpak and Discord launcher
│   ├── presence/presence.go         # Main monitoring loop and game state
│   ├── ui/tray.go                   # GTK System Tray native UI
│   └── steam/steam.go               # Steam AppID lookup
├── gnome-extension/                 # GNOME Shell extension source (reference copy)
├── config/
│   └── app_settings.json            # Base Application settings
└── lang/                            # Locale translations (en.json, es.json, etc.)
```

## Adding Browser Support

The monitor currently only targets the official GeForce NOW Flatpak container (`GeForceNOW`). To support playing GeForce NOW via Chrome/Edge browsers:
1. Modify `internal/detector/detector.go` to also scan for browser windows containing "GeForce NOW" in the title.
2. The GNOME Shell extension already returns all matching windows — update the `filter()` in `extension.js` to include `chrome` or `chromium` in the Wayland `WM_CLASS` check.

## Adding a New Compositor

To support a new Wayland compositor (e.g., Sway, River):
1. Add a new `getGFNTitle<Compositor>()` method in `detector.go`.
2. Use the compositor's IPC/D-Bus to query window titles.
3. Add it to the fallback chain in `getGFNWindowTitle()`.

## Custom Game Overrides

This application leverages a completely stateless Zero-Auth pipeline. There are no local databases to configure. If a game fails to map correctly to a Steam/GOG hero image, or fails to get picked up by Discord's Native App registry, you can right-click the System Tray icon and select **Force Game Name...** to manually override the title sent to Discord.
