//go:build js && wasm

package login

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/draw"
	"image/png"

	"github.com/kivutar/goro/client"
	xdraw "golang.org/x/image/draw"
)

// titleWebBackgroundData encodes the login background (single image or the
// 12-tile grid) as a one-time PNG data URL for the DOM layers.
func (m *Screen) titleWebBackgroundData(ctx client.Context) string {
	width, height := ctx.ScreenSize()
	if width <= 0 || height <= 0 || !m.bgLoaded {
		return ""
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(dst, dst.Bounds(), image.Black, image.Point{}, draw.Src)
	switch {
	case m.background != nil:
		scaleDraw(dst, m.background.RGBA())
	case len(m.bgTiles) == 12:
		cellW := float64(width) / 4
		cellH := float64(height) / 3
		for i, tile := range m.bgTiles {
			if tile == nil {
				continue
			}
			cell := image.Rect(
				int(float64(i%4)*cellW), int(float64(i/4)*cellH),
				int(float64(i%4+1)*cellW), int(float64(i/4+1)*cellH))
			xdraw.ApproxBiLinear.Scale(dst, cell, tile.RGBA(), tile.Bounds(), xdraw.Over, nil)
		}
	default:
		return ""
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func scaleDraw(dst *image.RGBA, src *image.RGBA) {
	b := src.Bounds()
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
}
