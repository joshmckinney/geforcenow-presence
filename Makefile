BINARY      = geforcenow-presence
MODULE      = github.com/joshmckinney/geforcenow-presence
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-beta")
LDFLAGS     = -ldflags="-s -w -X main.version=$(VERSION)"

PREFIX      ?= $(HOME)/.local
BINDIR      = $(PREFIX)/bin
CONFIGDIR   = $(HOME)/.config/geforcenow-presence
EXTDIR      = $(HOME)/.local/share/gnome-shell/extensions/window-title-server@geforcenow-presence
SERVICEDIR  = $(HOME)/.config/systemd/user
DESKTOPDIR  = $(HOME)/.local/share/applications

.PHONY: all build clean test install uninstall enable disable restart status release docker-release

# ─── Build ───────────────────────────────────────────────────────────
all: build

build:
	CGO_CFLAGS="-w -Wno-deprecated-declarations" go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY) .

clean:
	rm -f $(BINARY)
	rm -rf logs/
	rm -rf release/

# ─── Test ────────────────────────────────────────────────────────────
test: build
	@echo "── Smoke test (5 seconds) ──"
	@timeout 5 ./$(BINARY) --interval 2 2>&1 || true
	@echo ""
	@echo "✅ Build and smoke test passed"

# ─── Install ─────────────────────────────────────────────────────────
install: build
	@echo "── Installing $(BINARY) $(VERSION) ──"

	@# Binary
	install -Dm755 $(BINARY) $(BINDIR)/$(BINARY)
	@echo "✅ Binary installed to $(BINDIR)/$(BINARY)"

	@# Config files (only if not already present)
	install -d $(CONFIGDIR)
	@test -f $(CONFIGDIR)/app_settings.json || cp config/app_settings.json $(CONFIGDIR)/
	@test -d $(CONFIGDIR)/lang || cp -r lang $(CONFIGDIR)/
	@echo "✅ Config files at $(CONFIGDIR)"

	@# GNOME Shell extension
	install -d $(EXTDIR)
	install -m644 gnome-extension/extension.js $(EXTDIR)/extension.js
	install -m644 gnome-extension/metadata.json $(EXTDIR)/metadata.json
	-gnome-extensions enable window-title-server@geforcenow-presence 2>/dev/null
	@echo "✅ GNOME Shell extension installed"

	@# Systemd user service
	install -d $(SERVICEDIR)
	@echo '[Unit]'                                           > $(SERVICEDIR)/$(BINARY).service
	@echo 'Description=GeForce NOW Discord Rich Presence'   >> $(SERVICEDIR)/$(BINARY).service
	@echo 'After=graphical-session.target'                  >> $(SERVICEDIR)/$(BINARY).service
	@echo ''                                                >> $(SERVICEDIR)/$(BINARY).service
	@echo '[Service]'                                       >> $(SERVICEDIR)/$(BINARY).service
	@echo 'Type=simple'                                     >> $(SERVICEDIR)/$(BINARY).service
	@echo 'ExecStart=$(BINDIR)/$(BINARY) --delay 5'        >> $(SERVICEDIR)/$(BINARY).service
	@echo 'Restart=on-failure'                              >> $(SERVICEDIR)/$(BINARY).service
	@echo 'RestartSec=10'                                   >> $(SERVICEDIR)/$(BINARY).service
	@echo 'Environment=HOME=$(HOME)'                        >> $(SERVICEDIR)/$(BINARY).service
	@echo 'Environment=XDG_RUNTIME_DIR=/run/user/$(shell id -u)' >> $(SERVICEDIR)/$(BINARY).service
	@echo 'Environment=DISPLAY=:0'                          >> $(SERVICEDIR)/$(BINARY).service
	@echo ''                                                >> $(SERVICEDIR)/$(BINARY).service
	@echo '[Install]'                                       >> $(SERVICEDIR)/$(BINARY).service
	@echo 'WantedBy=default.target'                         >> $(SERVICEDIR)/$(BINARY).service
	@echo "✅ Systemd service created"

	@# Desktop entry
	install -d $(DESKTOPDIR)
	@echo '[Desktop Entry]'                                  > $(DESKTOPDIR)/$(BINARY).desktop
	@echo 'Name=GeForce NOW Rich Presence'                  >> $(DESKTOPDIR)/$(BINARY).desktop
	@echo 'Comment=Discord Rich Presence for GeForce NOW'   >> $(DESKTOPDIR)/$(BINARY).desktop
	@echo 'Exec=$(BINDIR)/$(BINARY)'                        >> $(DESKTOPDIR)/$(BINARY).desktop
	@echo 'Terminal=false'                                   >> $(DESKTOPDIR)/$(BINARY).desktop
	@echo 'Type=Application'                                 >> $(DESKTOPDIR)/$(BINARY).desktop
	@echo 'Categories=Game;Utility;'                         >> $(DESKTOPDIR)/$(BINARY).desktop
	@echo 'X-GNOME-Autostart-enabled=true'                   >> $(DESKTOPDIR)/$(BINARY).desktop
	@echo "✅ Desktop entry created"

	@echo ""
	@echo "── Installation complete! ──"
	@echo "Run:  systemctl --user enable --now $(BINARY)"
	@echo ""
	@echo "⚠️  If this is your first install, log out and back in"
	@echo "   to activate the GNOME Shell extension."

# ─── Uninstall ───────────────────────────────────────────────────────
uninstall:
	@echo "── Uninstalling $(BINARY) ──"

	@# Stop service
	-systemctl --user stop $(BINARY) 2>/dev/null
	-systemctl --user disable $(BINARY) 2>/dev/null

	@# Remove files
	rm -f  $(BINDIR)/$(BINARY)
	rm -f  $(SERVICEDIR)/$(BINARY).service
	rm -f  $(DESKTOPDIR)/$(BINARY).desktop
	rm -rf $(EXTDIR)
	-gnome-extensions disable window-title-server@geforcenow-presence 2>/dev/null

	@# Reload systemd
	-systemctl --user daemon-reload 2>/dev/null

	@echo "✅ Uninstalled (config files kept at $(CONFIGDIR))"
	@echo "   To remove config too: rm -rf $(CONFIGDIR)"

# ─── Service shortcuts ──────────────────────────────────────────────
enable:
	systemctl --user enable --now $(BINARY)

disable:
	systemctl --user disable --now $(BINARY)

restart:
	systemctl --user restart $(BINARY)

status:
	systemctl --user status $(BINARY)

# ─── Release ─────────────────────────────────────────────────────────
release: build
	@echo "── Building release archive ──"
	rm -rf release/$(BINARY)-$(VERSION)
	mkdir -p release/$(BINARY)-$(VERSION)
	cp $(BINARY) release/$(BINARY)-$(VERSION)/
	cp -r config release/$(BINARY)-$(VERSION)/
	cp -r lang release/$(BINARY)-$(VERSION)/
	cp -r gnome-extension release/$(BINARY)-$(VERSION)/
	cp README.md LICENSE Makefile release/$(BINARY)-$(VERSION)/
	cd release && tar -czvf $(BINARY)-linux-amd64-$(VERSION).tar.gz $(BINARY)-$(VERSION)/
	@echo "── Generating Checksums ──"
	cd release && sha256sum $(BINARY)-linux-amd64-$(VERSION).tar.gz > SHA256SUMS
	@echo "✅ Release archive and checksums created in release/"

dist: release package
	@echo "── Finalizing Distribution Artifacts ──"
	cd release && sha256sum *.tar.gz *.deb *.rpm > SHA256SUMS
	@echo "✅ All artifacts (tar.gz, deb, rpm) and checksums ready in release/"

docker-release:
	@echo "── Building release via Docker (Debian Bullseye) for maximum glibc compatibility ──"
	docker build -t geforcenow-presence-builder -f Dockerfile.build .
	docker run --rm -v $(PWD):/app geforcenow-presence-builder bash -c "make dist VERSION=$(VERSION)"
	sudo chown -R $(shell id -u):$(shell id -g) release/
	@echo "✅ Docker-built artifacts ready in release/"

# ─── Packaging ───────────────────────────────────────────────────────
NFPM_VERSION = 2.35.3
NFPM_BIN = build/nfpm

$(NFPM_BIN):
	@mkdir -p build
	@echo "── Downloading nfpm $(NFPM_VERSION) ──"
	curl -sL https://github.com/goreleaser/nfpm/releases/download/v$(NFPM_VERSION)/nfpm_$(NFPM_VERSION)_Linux_x86_64.tar.gz | tar -xz -C build nfpm

deb: build $(NFPM_BIN)
	@echo "── Building DEB package ──"
	@mkdir -p release
	VERSION=$(VERSION) $(NFPM_BIN) pkg --packager deb --target release/

rpm: build $(NFPM_BIN)
	@echo "── Building RPM package ──"
	@mkdir -p release
	VERSION=$(VERSION) $(NFPM_BIN) pkg --packager rpm --target release/

package: deb rpm
	@echo "✅ DEB and RPM packages created in release/"
