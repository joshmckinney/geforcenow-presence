package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	latestReleaseURL = "https://api.github.com/repos/joshmckinney/geforcenow-presence/releases/latest"
	releasesPageURL  = "https://github.com/joshmckinney/geforcenow-presence/releases"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// CheckForUpdate queries GitHub for the latest release tag and compares it to currentVersion.
// It returns the new version string if an update is available, otherwise empty string.
func CheckForUpdate(currentVersion string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(latestReleaseURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	if isNewer(current, latest) {
		return release.TagName, nil
	}

	return "", nil
}

// isNewer compares two version strings (e.g., "0.1.0-beta", "0.0.9").
// It handles numeric components and simple suffixes.
func isNewer(current, latest string) bool {
	if latest == "" || latest == current {
		return false
	}

	// Split by "-" to separate version from suffix (beta, alpha, etc.)
	p1 := strings.Split(current, "-")
	p2 := strings.Split(latest, "-")

	v1 := strings.Split(p1[0], ".")
	v2 := strings.Split(p2[0], ".")

	// Compare numeric parts
	for i := 0; i < len(v1) && i < len(v2); i++ {
		var n1, n2 int
		fmt.Sscanf(v1[i], "%d", &n1)
		fmt.Sscanf(v2[i], "%d", &n2)
		if n1 < n2 {
			return true
		}
		if n1 > n2 {
			return false
		}
	}

	// If numeric parts are equal, check if one has a suffix and the other doesn't
	// (e.g., 0.1.0-beta vs 0.1.0 -> 0.1.0 is newer)
	if len(v1) < len(v2) {
		return true
	}
	if len(p1) > 1 && len(p2) == 1 {
		return true // current has suffix, latest doesn't -> latest is newer
	}

	// If both have suffixes, fallback to string comparison (beta > alpha etc)
	if len(p1) > 1 && len(p2) > 1 {
		return p2[1] > p1[1]
	}

	return false
}

func GetReleasesURL() string {
	return releasesPageURL
}
