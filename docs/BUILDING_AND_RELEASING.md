# Building and Releasing

This document explains how to compile the application and generate portable distribution archives for end-users.

## Native Compilation
The application is designed to be built organically with zero external dependencies (other than GTK dev headers for the system tray).

1. Install Go 1.21+
2. Install C libraries:
   * **Debian/Ubuntu:** `sudo apt install libgtk-3-dev libayatana-appindicator3-dev`
   * **Fedora/RHEL:** `sudo dnf install gtk3-devel libayatana-appindicator-gtk3-devel`
3. Run `make build`

## Generating Release Artifacts (`make release`)

The application distribution pipeline is automated via the `Makefile`. 

To create a portable `.tar.gz` archive suitable for generic Linux distribution (including all binary assets, localization files, and install scripts):

```bash
make release
```

This will:
1. Compile the Go binary natively using your host's toolchain.
2. Bundle the binary, config stubs, translation files, and GNOME extensions into `release/geforcenow-presence-v1.0.0/`.
3. Compress the folder into a portable `geforcenow-presence-linux-amd64-v1.0.0.tar.gz` archive.
4. Calculate cryptographic `SHA256SUMS` to guarantee file integrity.

The generated archive will be located in the `release/` directory.

## 📦 Generating All Artifacts (`make dist`)

For a complete release, use the `dist` command. This consolidates the source tarball, DEB package, and RPM package into the `release/` folder and generates a collective `SHA256SUMS` file.

```bash
# Build everything for release
make dist
```

This is the recommended command to run before tagging a release on GitHub.

## 🤖 Automated CI/CD

The project uses GitHub Actions for continuous quality assurance:
*   **Continuous Integration (`ci.yml`)**: Every push to `main` and all Pull Requests trigger a suite of quality checks including `go vet`, `golangci-lint`, `go test`, and a full `make build` across standard Ubuntu environments.
*   **Automated Releases (`release.yml`)**: Tagging a release (e.g., `git tag v1.0.0`) automatically triggers a full `make dist` and uploads the resulting tarball, DEB, and RPM packages directly to the GitHub Release page.

## 🏗️ Installation Methods: Local vs. System

The project supports two distinct ways to install and run the application. Choosing the right one depends on your preference for system management.

### 1. Local User Install (`make install`)
*   **Target**: The current user only.
*   **Location**: `~/.local/bin`, `~/.config`, `~/.local/share/gnome-shell/extensions`.
*   **Management**: Handled manually via the `Makefile` (`make uninstall`).
*   **Best for**: Developers or users who want to run the latest code without system-wide changes.
*   **Permissions**: Does not require `sudo`.

### 2. System-Wide Package (`.deb` / `.rpm`)
*   **Target**: All users on the system.
*   **Location**: `/usr/bin`, `/etc`, `/usr/share`.
*   **Management**: Handled by your OS package manager (`apt`, `dnf`, `software center`).
*   **Best for**: General users who want automatic updates and standard Linux lifecycle management.
*   **Permissions**: Requires `sudo` to install.

---

> **Note on Compatibility:** While we provide a `make docker-release` target to build inside a Debian Bullseye container for maximum glibc compatibility, native compilation on your target machine (or using the generated packages) is generally preferred for modern distributions.

