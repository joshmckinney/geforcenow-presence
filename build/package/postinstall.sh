#!/bin/bash
# postinstall.sh

# Reload systemd
systemctl --user daemon-reload || true

echo "Installation complete."
echo "1. Enable the GNOME extension in your 'Extensions' app (or gnome-extensions CLI)."
echo "   NOTE: You may need 'AppIndicator and KStatusNotifierItem Support' for the tray icon to appear."
echo "2. Enable the service to start automatically:"
echo "   systemctl --user enable --now geforcenow-presence"
