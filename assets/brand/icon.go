package brand

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
)

// Sizes are the launcher icon sizes written for the menu and window.
var Sizes = []int{16, 24, 32, 48, 64, 128, 256}

// PNG renders the launcher mark at size pixels with Catmull-Rom high-fidelity filtering.
func PNG(size int) ([]byte, error) {
	src, err := Image()
	if err != nil {
		return nil, err
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	renderIcon(dst, src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ImageAt renders the launcher mark as an image with Catmull-Rom filtering.
func ImageAt(size int) (image.Image, error) {
	src, err := Image()
	if err != nil {
		return nil, err
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	renderIcon(dst, src)
	return dst, nil
}

// WriteIcons writes portpilot.png into the hicolor apps directories under root.
func WriteIcons(root string) error {
	for _, size := range []int{48, 128, 256} {
		raw, err := PNG(size)
		if err != nil {
			return err
		}
		dir := filepath.Join(root, itoa(size)+"x"+itoa(size), "apps")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		path := filepath.Join(dir, "portpilot.png")
		if err := os.WriteFile(path, raw, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func renderIcon(dst *image.RGBA, src image.Image) {
	b := dst.Bounds()
	w, h := b.Dx(), b.Dy()
	// High quality bicubic downscaling directly from source without any white plates or borders
	draw.CatmullRom.Scale(dst, b, src, src.Bounds(), draw.Src, nil)

	// Smoothly round the corners with anti-aliasing transparency
	rad := float64(w) / 6.0
	if rad < 2 {
		return
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy float64
			if float64(x) < rad {
				dx = rad - float64(x)
			} else if float64(x) > float64(w)-rad-1 {
				dx = float64(x) - (float64(w) - rad - 1)
			}
			if float64(y) < rad {
				dy = rad - float64(y)
			} else if float64(y) > float64(h)-rad-1 {
				dy = float64(y) - (float64(h) - rad - 1)
			}
			if dx > 0 && dy > 0 {
				dist := math.Hypot(dx, dy)
				if dist > rad+0.5 {
					dst.Set(x, y, color.NRGBA{0, 0, 0, 0})
				} else if dist > rad-0.5 {
					alpha := 0.5 - (dist - rad)
					if alpha < 0 {
						alpha = 0
					} else if alpha > 1 {
						alpha = 1
					}
					c := dst.RGBAAt(x, y)
					c.A = uint8(float64(c.A) * alpha)
					dst.SetRGBA(x, y, c)
				}
			}
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
