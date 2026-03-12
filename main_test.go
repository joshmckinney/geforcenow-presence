package main

import "testing"

func TestVersion(t *testing.T) {
	if version != "0.1.0-beta" && version != "dev" {
		t.Errorf("Expected version 0.1.0-beta or dev, got %s", version)
	}
}
