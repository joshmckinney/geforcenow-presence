package main

import "testing"

func TestVersion(t *testing.T) {
	expected := "1.0.0"
	if version != expected {
		t.Errorf("Expected version %s, got %s", expected, version)
	}
}
