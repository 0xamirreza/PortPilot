package gui

import (
	"image"
	"image/color"
	"testing"
)

func TestIconCardinalsPacksARGB(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.NRGBA{R: 0x10, G: 0x20, B: 0x30, A: 0xFF})
	got := iconCardinals([]image.Image{img})
	if len(got) != 3 || got[0] != 1 || got[1] != 1 || got[2] != 0xFF102030 {
		t.Fatalf("%v", got)
	}
}
