# Building and Releasing

This document explains how to compile the application and generate portable distribution archives for end-users.

## Native Compilation
The application is designed to be built organically with zero external dependencies (other than GTK dev headers for the system tray).

1. Install Go 1.25+
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
2. Bundle the binary, config stubs, translation files, and GNOME extensions into `release/geforcenow-presence-v0.1.0-beta/`.
3. Compress the folder into a portable `geforcenow-presence-linux-amd64-v0.1.0-beta.tar.gz` archive.
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
*   **Continuous Integration (`lint.yml`, `test.yml`, `build.yml`)**: Quality assurance is split into granular stages. Every push to `main` and all Pull Requests trigger dedicated workflows for `lint` (vet/linting), `test` (unit tests), and `build` (compilation) to provide detailed feedback via individual status badges.
*   **Automated Releases (`release.yml`)**: Tagging a release (e.g., `git tag v0.1.0-beta`) automatically triggers a full `make dist` and uploads the resulting tarball, DEB, and RPM packages directly to the GitHub Release page.

## 🏷️ Versioning and Build Tags

The application uses dynamic versioning to ensure that every binary and package is traceable to a specific point in the Git history. This is handled via the `Makefile` and Go's `-ldflags`.

### How it works:
1. **Dynamic Detection**: The `Makefile` runs `git describe --tags --always --dirty` to determine the current version.
   - **Tagged Release**: If you are on a tag, the version is simply the tag name (e.g., `v0.2.0-beta`).
   - **Dev Build**: If you are between tags, it includes the number of commits since the last tag and the short hash (e.g., `v0.1.4-beta-2-g1f0a536`).
   - **Dirty State**: If you have unstaged changes, a `-dirty` suffix is appended, ensuring you know the build doesn't match a clean commit.
2. **Binary Injection**: This version string is baked into the binary during compilation using:
   ```bash
   go build -ldflags="-X main.version=$(VERSION)"
   ```
3. **Traceability**: You can always see which exact version is running by checking the logs or the "Version" field in the system tray.

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

