package skills

import (
	"image"
	"image/color"
	"math"
	"reflect"
	"testing"

	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestGraffitiBitmapAlphabet(t *testing.T) {
	atlas := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			atlas.SetRGBA(x, y, color.RGBA{R: uint8(y/32*8 + x/32), A: 255})
		}
	}
	texture := graffitiTexture(atlas, "Az9?!")
	y := 5 * 36 // native vertical placement for a five-character message
	for _, tc := range []struct{ x, glyph int }{{0, 0}, {32, 51}, {64, 63}, {96, 53}, {112, 54}} {
		if got := texture.RGBA().RGBAAt(tc.x, y); got.R != uint8(tc.glyph) || got.A != 255 {
			t.Fatalf("glyph at x=%d = %+v, want %d", tc.x, got, tc.glyph)
		}
	}
	if texture.RGBA().RGBAAt(128, y).A != 0 || texture.RGBA().RGBAAt(0, 0).A != 0 {
		t.Fatal("Scribble must keep the area outside its letters transparent")
	}
	// Server text can contain a full 80-byte field and unsupported glyphs.
	graffitiTexture(atlas, "Some long words followed by unknown bytes: \xff\xfe and punctuation. 0123456789 abcdefghij")
}

func TestGraffitiFollowsGroundTriangles(t *testing.T) {
	for _, tc := range []struct {
		name    string
		heights [4]float32
		grid    [3][3]float64
	}{
		{"raised diagonal", [4]float32{0, 0, 0, 4}, [3][3]float64{{0, 0, 0}, {0, 2, 2}, {0, 2, 4}}},
		{"lowered diagonal", [4]float32{0, 4, 4, 0}, [3][3]float64{{0, 2, 4}, {2, 0, 2}, {4, 2, 0}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := testutil.RogueSkillContext()
			ctx.World.GND = &res.GND{Width: 1, Height: 1, Cells: []res.GNDCell{{Heights: tc.heights}}}
			texture := render.NewImage(512, 512)
			writing := &groundGraffiti{X: 9, Y: 9}
			mesh := graffitiMesh(ctx, writing, texture)
			if mesh.VertexCount() != 16 {
				t.Fatalf("expected four quads covering one GND cell, got %d vertices", mesh.VertexCount())
			}
			vertices := reflect.ValueOf(mesh).Elem().FieldByName("vertices")
			for i := 0; i < vertices.Len(); i++ {
				v := vertices.Index(i)
				x, y := int(v.FieldByName("X").Float()), int(v.FieldByName("Z").Float())
				if got, want := v.FieldByName("Y").Float(), tc.grid[y][x]+.01; math.Abs(got-want) > .0001 {
					t.Fatalf("height at %d,%d = %g, want terrain height + lift = %g", x, y, got, want)
				}
			}
			ctx.World.GND.Cells[0].Top = -1
			if mesh := graffitiMesh(ctx, writing, texture); mesh.VertexCount() != 0 {
				t.Fatal("writing appeared over a missing ground surface")
			}
		})
	}
}
