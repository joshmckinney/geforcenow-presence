package detector

import "testing"

func TestParseNiriWindowsJSON(t *testing.T) {
	realSessionPayload := []byte(`[
		{"id":75,"title":"@Mark - Discord","app_id":"discord","pid":80914,"workspace_id":7,"is_focused":false},
		{"id":70,"title":"All Mail (157 unread)","app_id":"org.gnome.Evolution","pid":65459,"workspace_id":1,"is_focused":false},
		{"id":71,"title":"POSTAL 4: No Regerts on GeForce NOW","app_id":"GeForceNOW","pid":3428,"workspace_id":7,"is_focused":false}
	]`)

	got := parseNiriWindowsJSON(realSessionPayload)
	want := "POSTAL 4: No Regerts on GeForce NOW"
	if got != want {
		t.Fatalf("parseNiriWindowsJSON() = %q, want %q", got, want)
	}

	if cleaned := cleanTitle(got); cleaned != "POSTAL 4: No Regerts" {
		t.Fatalf("cleanTitle(%q) = %q, want %q", got, cleaned, "POSTAL 4: No Regerts")
	}
}

func TestParseNiriWindowsJSON_PrefersActiveGameWindow(t *testing.T) {
	multiWindowPayload := []byte(`[
		{"id":1,"title":"GeForce NOW","app_id":"GeForceNOW"},
		{"id":2,"title":"GeForce NOW","app_id":"geforcenow-presence"},
		{"id":3,"title":"Cyberpunk 2077® on GeForce NOW","app_id":"com.nvidia.geforcenow"}
	]`)

	got := parseNiriWindowsJSON(multiWindowPayload)
	want := "Cyberpunk 2077® on GeForce NOW"
	if got != want {
		t.Fatalf("parseNiriWindowsJSON() = %q, want %q", got, want)
	}

	if cleaned := cleanTitle(got); cleaned != "Cyberpunk 2077" {
		t.Fatalf("cleanTitle(%q) = %q, want %q", got, cleaned, "Cyberpunk 2077")
	}
}

func TestParseNiriWindowsJSON_IgnoresNonGFNAndInvalid(t *testing.T) {
	nonGFN := []byte(`[
		{"id":1,"title":"Some Game on GeForce NOW","app_id":"geforcenow-presence"},
		{"id":2,"title":"Firefox","app_id":"firefox"}
	]`)
	if got := parseNiriWindowsJSON(nonGFN); got != "" {
		t.Fatalf("expected empty title for non-GFN windows, got %q", got)
	}

	if got := parseNiriWindowsJSON([]byte(`not-json`)); got != "" {
		t.Fatalf("expected empty title for invalid JSON, got %q", got)
	}
}

func TestIsGNOME(t *testing.T) {
	t.Setenv("NIRI_SOCKET", "/run/user/1000/niri.wayland-1.sock")
	t.Setenv("XDG_SESSION_DESKTOP", "niri")
	t.Setenv("DESKTOP_SESSION", "niri")
	t.Setenv("XDG_CURRENT_DESKTOP", "niri:GNOME")
	if isGNOME() {
		t.Fatalf("expected isGNOME() = false when running on Niri with XDG_CURRENT_DESKTOP=niri:GNOME")
	}

	t.Setenv("NIRI_SOCKET", "")
	t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "")
	t.Setenv("XDG_SESSION_DESKTOP", "gnome")
	t.Setenv("DESKTOP_SESSION", "gnome")
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	if !isGNOME() {
		t.Fatalf("expected isGNOME() = true on GNOME session")
	}
}
