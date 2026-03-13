package metadata

import (
	"testing"
)

func TestFetchArtFallback(t *testing.T) {
	// Since we can't easily mock the external steam/gog packages without refactoring
	// them to interfaces (which might be overkill for this request), we verify
	// the fallback behavior for an unknown game.

	result := FetchArt("NonExistentGame123456789")
	if result != "geforce_now" {
		t.Errorf("Expected fallback 'geforce_now', got '%s'", result)
	}
}
