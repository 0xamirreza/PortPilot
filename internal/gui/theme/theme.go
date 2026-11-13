package theme

import (
	"image/color"

	"gioui.org/font"
	"gioui.org/font/opentype"
	"gioui.org/text"
	"gioui.org/widget/material"

	"github.com/0xamirreza/portpilot/assets/fonts"
)

const (
	Sans = font.Typeface("Liberation Sans")
	Mono = font.Typeface("Liberation Mono")
)

// Palette is the brutalist plate: hairline rules, no radius, one signal color.
type Palette struct {
	Void   color.NRGBA
	Plate  color.NRGBA
	Raised color.NRGBA
	Rule   color.NRGBA
	Ink    color.NRGBA
	Mute   color.NRGBA
	Signal color.NRGBA
	Stamp  color.NRGBA
	Olive  color.NRGBA
}

// Theme is the active palette plus the shared text shaper.
type Theme struct {
	Material *material.Theme
	Palette
	Dark bool
}

// Load reads the embedded Liberation faces. System fonts are not used.
func Load() (*Theme, error) {
	names := []string{
		"LiberationSans-Regular.ttf",
		"LiberationSans-Bold.ttf",
		"LiberationMono-Regular.ttf",
		"LiberationMono-Bold.ttf",
	}
	faces := make([]text.FontFace, 0, len(names))
	for _, name := range names {
		raw, err := fonts.Files.ReadFile(name)
		if err != nil {
			return nil, err
		}
		face, err := opentype.Parse(raw)
		if err != nil {
			return nil, err
		}
		faces = append(faces, text.FontFace{Font: face.Font(), Face: face})
	}
	m := material.NewTheme()
	m.Shaper = text.NewShaper(text.WithCollection(faces))
	m.Face = Sans
	m.TextSize = 14
	t := &Theme{Material: m}
	t.Use(true)
	return t, nil
}

// Use switches the dark or light plate without rebuilding the shaper.
func (t *Theme) Use(dark bool) {
	t.Dark = dark
	if dark {
		t.Palette = Palette{
			Void: hex(0x12110C), Plate: hex(0x1C1A14), Raised: hex(0x2A261C),
			Rule: hex(0x4A4336), Ink: hex(0xF3EBDD), Mute: hex(0x948C7C),
			Signal: hex(0xE10600), Stamp: hex(0xE2B340), Olive: hex(0xA3B56A),
		}
	} else {
		t.Palette = Palette{
			Void: hex(0xD5DBE1), Plate: hex(0xEEF1F4), Raised: hex(0xF7F8FA),
			Rule: hex(0x1C2126), Ink: hex(0x14181C), Mute: hex(0x52606A),
			Signal: hex(0xC4120A), Stamp: hex(0x8A5A00), Olive: hex(0x2C6B3C),
		}
	}
	t.Material.Palette = material.Palette{
		Bg: t.Void, Fg: t.Ink, ContrastBg: t.Ink, ContrastFg: t.Void,
	}
	t.Material.Face = Sans
}

func hex(v uint32) color.NRGBA {
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xFF}
}
