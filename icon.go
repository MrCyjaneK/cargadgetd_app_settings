package main

import (
	"bytes"
	"embed"
	"image"
	"image/draw"
	"image/png"
)

//go:embed icon.png
var iconFS embed.FS

var iconPix []byte
var iconW, iconH int

func loadIcon() {
	if iconPix != nil {
		return
	}
	raw, err := iconFS.ReadFile("icon.png")
	if err != nil {
		return
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return
	}
	rgba := fitIcon(img, 64)
	iconPix = append([]byte(nil), rgba.Pix...)
	iconW, iconH = rgba.Bounds().Dx(), rgba.Bounds().Dy()
}

// fitIcon samples src down to an n by n icon. The home grid stores a 64px glyph.
func fitIcon(src image.Image, n int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, n, n))
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw <= 0 || sh <= 0 {
		return dst
	}
	if sw == n && sh == n {
		draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
		return dst
	}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			dst.Set(x, y, src.At(b.Min.X+x*sw/n, b.Min.Y+y*sh/n))
		}
	}
	return dst
}
