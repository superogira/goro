// Command genicon renders the GorORG35 app-list icon (GorORG35.png).
//
// The RG35XX StockOS frontend shows a PNG named after the launcher script
// (GorORG35.sh -> GorORG35.png) next to it in Roms/APPS. Everything is
// drawn procedurally — no font or asset dependencies — so the icon
// regenerates anywhere Go runs.
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

const size = 256

func main() {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Background: vertical navy-to-steel gradient with rounded corners
	// and a gold frame, echoing the classic RO login window.
	for y := 0; y < size; y++ {
		t := float64(y) / size
		for x := 0; x < size; x++ {
			if !inRoundedCorner(x, y, 40) {
				continue
			}
			c := lerp(color.RGBA{18, 28, 58, 255}, color.RGBA{52, 84, 130, 255}, t)
			img.Set(x, y, c)
		}
	}
	// Soft vignette so the center art pops.
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if !inRoundedCorner(x, y, 40) {
				continue
			}
			d := distToCenter(x, y) / (size * 0.72)
			if d > 1 {
				d = 1
			}
			img.Set(x, y, darken(img.At(x, y), d*0.45))
		}
	}
	// Gold frame (double line).
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			edge := frameDist(x, y)
			if edge < 3 && inRoundedCorner(x, y, 40) {
				img.Set(x, y, color.RGBA{212, 175, 55, 255})
			} else if edge >= 3 && edge < 5 && inRoundedCorner(x, y, 36) {
				img.Set(x, y, color.RGBA{160, 124, 32, 255})
			}
		}
	}

	drawSword(img)
	drawTitle(img)

	out, err := os.Create("GorORG35.png")
	if err != nil {
		panic(err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		panic(err)
	}
}

func inRoundedCorner(x, y, r int) bool {
	corners := [][2]int{{r, r}, {size - 1 - r, r}, {r, size - 1 - r}, {size - 1 - r, size - 1 - r}}
	for _, c := range corners {
		cx, cy := c[0], c[1]
		inBox := (x < r && y < r) || (x >= size-r && y < r) ||
			(x < r && y >= size-r) || (x >= size-r && y >= size-r)
		if inBox {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy > r*r {
				return false
			}
		}
	}
	return true
}

func frameDist(x, y int) int {
	m := x
	if y < m {
		m = y
	}
	if size-1-x < m {
		m = size - 1 - x
	}
	if size-1-y < m {
		m = size - 1 - y
	}
	return m
}

func distToCenter(x, y int) float64 {
	dx, dy := float64(x-size/2), float64(y-size/2)
	return sqrtApprox(dx*dx + dy*dy)
}

func sqrtApprox(v float64) float64 {
	if v <= 0 {
		return 0
	}
	g := v
	for i := 0; i < 24; i++ {
		g = (g + v/g) / 2
	}
	return g
}

func lerp(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		255,
	}
}

func darken(c color.Color, f float64) color.Color {
	r, g, b, _ := c.RGBA()
	return color.RGBA{uint8(float64(r>>8) * (1 - f)), uint8(float64(g>>8) * (1 - f)), uint8(float64(b>>8) * (1 - f)), 255}
}

// drawSword paints a stylized upward sword centered above the title.
func drawSword(img *image.RGBA) {
	cx := size / 2

	// Blade: tapering white-blue slab from y=48 down to y=170.
	for y := 48; y <= 170; y++ {
		t := float64(y-48) / 122
		half := 7 - int(4*t) // 7px half-width at tip -> 3 at guard
		for x := cx - half; x <= cx+half; x++ {
			edge := x == cx-half || x == cx+half
			c := color.RGBA{214, 228, 244, 255} // blade body
			if edge {
				c = color.RGBA{140, 165, 200, 255} // shaded edge
			}
			if y < 56 {
				c = color.RGBA{255, 255, 255, 255} // bright tip
			}
			img.Set(x, y, c)
		}
	}
	// Center ridge highlight.
	for y := 50; y < 168; y++ {
		img.Set(cx, y, color.RGBA{255, 255, 255, 255})
	}

	// Crossguard: gold bar with flared ends.
	guardY := 171
	for x := cx - 34; x <= cx+34; x++ {
		depth := 4
		if abs(x-cx) > 26 {
			depth = 7 // flared tips
		}
		for y := guardY; y < guardY+depth; y++ {
			c := color.RGBA{212, 175, 55, 255}
			if y == guardY || (abs(x-cx) > 26 && x == cx-34+((y-guardY)*0)) {
				c = color.RGBA{255, 216, 110, 255}
			}
			img.Set(x, y, c)
		}
	}

	// Grip: dark leather with wraps.
	for y := 175; y <= 196; y++ {
		for x := cx - 4; x <= cx+4; x++ {
			c := color.RGBA{74, 50, 32, 255}
			if (y-175)%6 < 2 {
				c = color.RGBA{96, 68, 44, 255} // wrap highlight
			}
			img.Set(x, y, c)
		}
	}

	// Pommel: gold diamond.
	for y := 197; y <= 210; y++ {
		t := abs(y - 203)
		for x := cx - (7 - t); x <= cx+(7-t); x++ {
			img.Set(x, y, color.RGBA{212, 175, 55, 255})
		}
	}
	img.Set(cx, 203, color.RGBA{255, 216, 110, 255})
}

// drawTitle stamps "GORO" in block pixels under the sword.
func drawTitle(img *image.RGBA) {
	glyphs := map[rune][5]uint16{
		'G': {
			0b01111,
			0b10000,
			0b10111,
			0b10001,
			0b01110,
		},
		'O': {
			0b01110,
			0b10001,
			0b10001,
			0b10001,
			0b01110,
		},
		'R': {
			0b11110,
			0b10001,
			0b11110,
			0b10100,
			0b10010,
		},
	}
	word := "GORO"
	cell := 9
	startX := size/2 - len(word)*(5*cell+cell)/2 + cell/2
	startY := 218
	for i, ch := range word {
		g := glyphs[ch]
		ox := startX + i*(5*cell+cell)
		for row := 0; row < 5; row++ {
			for col := 0; col < 5; col++ {
				if g[row]&(1<<(4-col)) == 0 {
					continue
				}
				for dy := 0; dy < cell; dy++ {
					for dx := 0; dx < cell; dx++ {
						x := ox + col*cell + dx
						y := startY + row*cell + dy
						if x < 0 || y < 0 || x >= size || y >= size {
							continue
						}
						edge := dy == 0 || dx == 0
						if edge {
							img.Set(x, y, color.RGBA{255, 232, 150, 255})
						} else {
							img.Set(x, y, color.RGBA{222, 186, 80, 255})
						}
					}
				}
			}
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
