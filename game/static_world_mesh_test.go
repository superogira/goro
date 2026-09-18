package game

import (
	"image/color"
	"testing"

	"github.com/kivutar/goro/res"
)

func testGNDWithLightmapAlpha(width, height int, alphaAt func(x, y int) uint8) *res.GND {
	gnd := testGNDWithTopHeights(width, height, func(x, y int) [4]float32 {
		return [4]float32{}
	})
	shadowed := res.GNDLightmap{}
	lit := res.GNDLightmap{}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			shadowed.Alpha[y][x] = 128
			lit.Alpha[y][x] = 255
		}
	}
	gnd.Lightmaps = []res.GNDLightmap{shadowed, lit}
	gnd.Surfaces = make([]res.GNDSurface, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			id := 1
			if alphaAt(x, y) == 128 {
				id = 0
			}
			gnd.Surfaces[x+y*width] = res.GNDSurface{TextureID: 0, LightmapID: id}
		}
	}
	for i := range gnd.Cells {
		gnd.Cells[i].Top = i
	}
	return gnd
}

func TestBuildGNDGroundLightmapBlursTileBoundaries(t *testing.T) {
	// Left half of the map shadowed at alpha 128, right half lit at 255.
	gnd := testGNDWithLightmapAlpha(8, 8, func(x, y int) uint8 {
		if x < 4 {
			return 128
		}
		return 255
	})
	lightmap := buildGNDGroundLightmap(gnd)
	if lightmap.image == nil {
		t.Fatal("ground lightmap image is nil")
	}
	bounds := lightmap.image.Bounds()
	if bounds.Dx() != 8*gndLightmapTexelsPerTile || bounds.Dy() != 8*gndLightmapTexelsPerTile {
		t.Fatalf("lightmap bounds = %v, want %dx%d", bounds, 8*gndLightmapTexelsPerTile, 8*gndLightmapTexelsPerTile)
	}
	// Far inside each half keeps its level after the blur.
	deepShadow := lightmap.image.RGBA().At(1*gndLightmapTexelsPerTile, 4*gndLightmapTexelsPerTile)
	deepLit := lightmap.image.RGBA().At(7*gndLightmapTexelsPerTile, 4*gndLightmapTexelsPerTile)
	if got := colorAlpha(deepShadow); got > 150 {
		t.Fatalf("deep shadow alpha = %d, want <= 150", got)
	}
	if got := colorAlpha(deepLit); got < 220 {
		t.Fatalf("deep lit alpha = %d, want >= 220", got)
	}
	// On the boundary the blur must produce intermediate steps rather than a
	// hard jump between the two halves.
	transitioned := false
	previous := -1
	for x := 3 * gndLightmapTexelsPerTile; x < 5*gndLightmapTexelsPerTile; x++ {
		got := colorAlpha(lightmap.image.RGBA().At(x, 4*gndLightmapTexelsPerTile))
		if previous >= 0 && got != previous {
			transitioned = true
		}
		previous = got
	}
	if !transitioned {
		t.Fatal("lightmap boundary shows no intermediate blur steps")
	}
}

func TestGroundLightmapTileUVsCoverAdjacentTiles(t *testing.T) {
	left := groundLightmapTileUVs(2, 3)
	right := groundLightmapTileUVs(3, 3)
	if left[1].u != right[0].u || left[2].u != right[3].u {
		t.Fatalf("neighboring tiles do not share UV edges: %v vs %v", left, right)
	}
	if left[0].u != left[3].u || left[1].u != left[2].u || left[0].v != left[1].v || left[2].v != left[3].v {
		t.Fatalf("tile UVs are not an axis-aligned quad: %v", left)
	}
}

func colorAlpha(c color.Color) int {
	_, _, _, a := c.RGBA()
	return int(a >> 8)
}
