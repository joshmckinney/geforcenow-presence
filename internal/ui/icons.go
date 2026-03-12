package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
)

var (
	iconGreen  []byte
	iconYellow []byte
	iconRed    []byte
)

func init() {
	// Defaults will be overridden by RebuildIcons on startup
	iconGreen = GenerateCircleIcon(color.RGBA{R: 46, G: 204, B: 113, A: 255})
	iconYellow = GenerateCircleIcon(color.RGBA{R: 241, G: 196, B: 15, A: 255})
	iconRed = GenerateCircleIcon(color.RGBA{R: 231, G: 76, B: 60, A: 255})
}

// RebuildIcons updates the icon byte arrays from the provided color strings.
func RebuildIcons(colors map[string]string) {
	if c, ok := colors["playing"]; ok {
		iconGreen = GenerateCircleIcon(parseColor(c))
	}
	if c, ok := colors["waiting"]; ok {
		iconYellow = GenerateCircleIcon(parseColor(c))
	}
	if c, ok := colors["error"]; ok {
		iconRed = GenerateCircleIcon(parseColor(c))
	}
}

func GenerateCircleIcon(c color.Color) []byte {
	size := 64
	radius := 28
	center := size / 2

	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Draw transparent background is default
	// Draw circle
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := x - center
			dy := y - center
			if dx*dx+dy*dy <= radius*radius {
				img.Set(x, y, c)
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func parseColor(s string) color.RGBA {
	s = strings.TrimSpace(s)
	// Handle rgb(r, g, b)
	if strings.HasPrefix(s, "rgb") {
		var r, g, b uint8
		s = strings.TrimPrefix(s, "rgba(")
		s = strings.TrimPrefix(s, "rgb(")
		s = strings.TrimSuffix(s, ")")
		_, _ = fmt.Sscanf(strings.ReplaceAll(s, ",", " "), "%d %d %d", &r, &g, &b)
		return color.RGBA{R: r, G: g, B: b, A: 255}
	}

	// Handle Hex
	hex := strings.TrimPrefix(s, "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return color.RGBA{A: 255}
	}
	var r, g, b uint8
	_, _ = fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return color.RGBA{R: r, G: g, B: b, A: 255}
}
