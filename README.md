<div align="center">
  <h1>🎮 GeForce NOW Rich Presence for Discord — Linux</h1>
  <p>
    <strong>Show your real game on Discord while playing on GeForce NOW — automatically, on Linux.</strong>
  </p>
  <p>
    A compiled Go binary that monitors the <a href="https://international.download.nvidia.com/GFNLinux/flatpak/geforcenow.flatpakrepo">GeForce NOW</a> Flatpak,
    detects the game you're playing via the window title, and updates your Discord Rich Presence
    with the correct game name, artwork, and elapsed time.
  </p>

  [![Latest Release](https://img.shields.io/github/v/release/joshmckinney/geforcenow-presence?style=for-the-badge&color=00C853&logo=github)](https://github.com/joshmckinney/geforcenow-presence/releases/latest)
  [![Downloads](https://img.shields.io/github/downloads/joshmckinney/geforcenow-presence/total?style=for-the-badge&color=2962FF&logo=github)](https://github.com/joshmckinney/geforcenow-presence/releases)
  <br>
  [![CI](https://github.com/joshmckinney/geforcenow-presence/actions/workflows/ci.yml/badge.svg)](https://github.com/joshmckinney/geforcenow-presence/actions/workflows/ci.yml)
  <br>
  [![Test](https://github.com/joshmckinney/geforcenow-presence/actions/workflows/test.yml/badge.svg)](https://github.com/joshmckinney/geforcenow-presence/actions/workflows/test.yml)
  [![Build](https://github.com/joshmckinney/geforcenow-presence/actions/workflows/build.yml/badge.svg)](https://github.com/joshmckinney/geforcenow-presence/actions/workflows/build.yml)
  [![Lint](https://github.com/joshmckinney/geforcenow-presence/actions/workflows/lint.yml/badge.svg)](https://github.com/joshmckinney/geforcenow-presence/actions/workflows/lint.yml)
</div>

---

## ✨ Features

- 🐧 **Native Linux support** — compiled Go binary, zero runtime dependencies
- 🎮 **Automatic game detection** — reads the GeForce NOW window title via GNOME Shell extension D-Bus
- 🖼️ **Zero-Auth Metadata Pipeline** — queries public Steam and GOG APIs to push live HTTP box-art URLs to Discord natively without developer keys
- 🌟 **Native Discord Apps Tracking** — dynamically syncs with Discord's 22,000+ app database. Fuzzy matches game strings directly to official Client IDs to natively display the game on your profile.
- 🚀 **Auto-start support** — systemd user service for background operation on login (toggleable via UI)
- ⚙️ **Persistent Configuration** — Polling intervals, startup delays, and language settings are saved and easily editable via the System Tray.
- 🎨 **Custom Status Colors** — Personalize your tray icon's "Playing", "Idle", and "Error" states via a built-in color picker.
- 🛡️ **Optional Game History** — Capture Geforce NOW sessions in your Discord 30-day activity record via a specialized dummy-process launcher (disabled by default).
- 🚀 **Update Checker** — Automatic background checks for new releases on startup + manual check button.
- 📂 **Quick Access** — Open logs or configuration folders directly from the tray.
- 🧩 **Multi-compositor support** — GNOME (Wayland), X11 (xprop/xdotool), Hyprland, with fallback chain
- 🔒 **Single instance** — lock file prevents duplicate processes
- 🌐 **i18n** — locale detection with language files in `lang/`

---

## 📋 Requirements

| Requirement | Details |
|:---|:---|
| **OS** | Linux (tested on Fedora 43, should work on any distro with GNOME 45+) |
| **GeForce NOW** | [Official Nvidia Flatpak](https://international.download.nvidia.com/GFNLinux/flatpak/geforcenow.flatpakrepo) (`com.nvidia.geforcenow`) |
| **Discord** | Native install (RPM/DEB) or Flatpak |
| **Go** | 1.25+ (build only — not needed to run the binary) |
| **Build Dependencies** | `libayatana-appindicator3-dev` (Debian/Ubuntu) or `libayatana-appindicator-gtk3-devel` (Fedora/RHEL)<br>`libgtk-3-dev` (Debian/Ubuntu) or `gtk3-devel` (Fedora/RHEL) |
| **Desktop** | GNOME on Wayland (primary), X11, Hyprland (experimental) |

> **Note:** Running GeForce NOW via Chrome/Edge browser is **not currently supported** — only the official Flatpak app. Browser support could be added by detecting the browser window title similarly.

---

## 🚀 Quick Start

```bash
# Clone and build
git clone https://github.com/joshmckinney/geforcenow-presence.git
cd geforcenow-presence
make install
```

After the first run, **log out and back in** to activate the GNOME Shell extension.

---

## 📥 Installation

### 📦 Native Packages (Recommended)
Download the latest release for your distribution:
- **Debian / Ubuntu / Mint**: `.deb` package
- **Fedora / RHEL / openSUSE**: `.rpm` package
- **Generic Linux**: `.tar.gz` bundle

Install using your package manager (e.g., `sudo dnf install ./geforcenow-presence.rpm`).

### 🛠️ Build from Source (`make install`)

`make install` sets up everything needed for production use:

| Component | Location | Purpose |
|:---|:---|:---|
| **Binary** | `~/.local/bin/geforcenow-presence` | The compiled Go executable |
| **Config** | `~/.config/geforcenow-presence/` | `app_settings.json` and local cache files, `lang/` |
| **GNOME Extension** | `~/.local/share/gnome-shell/extensions/window-title-server@geforcenow-presence/` | Reads Wayland-native window titles via D-Bus |
| **Systemd Service** | `~/.config/systemd/user/geforcenow-presence.service` | Runs the binary as a background daemon on login |
| **Desktop Entry** | `~/.local/share/applications/geforcenow-presence.desktop` | Shows in application launcher |

After installing, enable auto-start:

```bash
# Start now + enable on login
make enable
# or equivalently:
systemctl --user enable --now geforcenow-presence
```

### First-Time Setup

On GNOME Wayland, the GNOME Shell extension needs a session restart to load:

1. Run `make install`
2. **Log out and back in** (or restart GNOME Shell)
3. Run `make enable`
4. Launch a game on GeForce NOW — your Discord status will update automatically

---

## 🗑️ Uninstalling (`make uninstall`)

```bash
make uninstall
```

This cleanly removes:
- ✅ Stops and disables the systemd service
- ✅ Removes the binary from `~/.local/bin/`
- ✅ Removes the systemd service file
- ✅ Removes the desktop entry
- ✅ Removes the GNOME Shell extension
- ✅ Reloads systemd daemon

Config files at `~/.config/geforcenow-presence/` are **preserved**. To remove those too:

```bash
rm -rf ~/.config/geforcenow-presence
```

---

## ⚙️ Usage

```bash
# Run in foreground (useful for debugging)
./geforcenow-presence

# With options (CLI flags override persistent settings)
./geforcenow-presence --delay 10    # wait 10s before starting
./geforcenow-presence --interval 5  # poll every 5 seconds

# Tray Management
The System Tray provides easy access to:
- **Interval**: Current polling rate (Click to change to any integer)
- **Delay**: Current startup delay (Click to change to any integer)
- **Auto-start on Login**: Enable/Disable the background service
- **Language**: Switch between available translations
- **Custom Status Colors**: Choose colors for app states via Zenity
- **Check for Updates**: Manually trigger a version check
- **Open Config Folder**: Quickly access your local settings

# Service management
make status     # check service status
make restart    # restart the service
make disable    # stop and disable auto-start
```

---

## 📚 Documentation

Detailed technical information for developers and advanced users has been moved to our `docs/` directory:

- [**Architecture & Design**](docs/ARCHITECTURE.md) - Explains the D-Bus flow, Zero-Auth metadata pipeline, and Discord IPC mechanisms.
- [**Building and Releasing**](docs/BUILDING_AND_RELEASING.md) - Explains how to compile from source, generate generic Linux `.tar.gz` distribution artifacts, and build `.deb`/`.rpm` packages.
- [**Extending & Modifying**](docs/CONTRIBUTING.md) - Displays the project source tree and explains how to add support for web browsers, new Wayland compositors, and custom game overrides.

---

## 🤝 Support & Contributing

If you encounter issues or have feature requests for this Linux Go rewrite, please open an issue on [the GitHub repository](https://github.com/joshmckinney/geforcenow-presence/issues).

Contributions are welcome! Areas where help is appreciated:

- 🖥️ **More compositors** — Sway, KDE Plasma (KWin), River, etc.
- 🧪 **Testing** — on different distros and desktop environments

---

## 📜 License & Credits

This project is licensed under the **MIT License** — see [LICENSE](LICENSE) for details.

This is a Linux fork of the original [GeForce NOW Rich Presence](https://github.com/KarmaDevz/GeForce-NOW-Rich-Presence) by [KarmaDevz](https://github.com/KarmaDevz), completely rewritten from Python to Go for native Linux/Wayland support by [Josh McKinney](https://github.com/joshmckinney). 

---

## ⚠️ Disclaimer

This software is **not affiliated with** NVIDIA, GeForce NOW, Discord, or Valve. All product names and logos are property of their respective owners. This application interacts with Discord's Rich Presence IPC API — use at your own discretion. See [LICENSE](LICENSE) for full terms.
