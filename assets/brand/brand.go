// Package brand holds the PortPilot mark.
package brand

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/jpeg"
)

//go:embed logo.jpeg
var JPEG []byte

// Image decodes the embedded logo.
func Image() (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(JPEG))
	return img, err
}
