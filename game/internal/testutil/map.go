package testutil

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func PathGAT(width, height int, blocked map[worldstate.WalkStep]bool) *res.GAT {
	gat := &res.GAT{
		Width:  width,
		Height: height,
		Cells:  make([]res.GATCell, width*height),
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cellType := res.GATTypeWalkable | res.GATTypeSnipable
			if blocked[worldstate.WalkStep{X: x, Y: y}] {
				cellType = res.GATTypeNone
			}
			gat.Cells[y*width+x] = res.GATCell{Type: cellType}
		}
	}
	return gat
}

func FlatWalkableGAT(width, height int) *res.GAT {
	gat := &res.GAT{
		Width:  width,
		Height: height,
		Cells:  make([]res.GATCell, width*height),
	}
	for i := range gat.Cells {
		gat.Cells[i] = res.GATCell{Type: res.GATTypeWalkable}
	}
	return gat
}

func MapTextureContext(t *testing.T) client.Context {
	t.Helper()
	root := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, B: 255, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{G: 255, A: 255})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	names := []string{"ground.png", "model.png"}
	for i := 0; i < 32; i++ {
		names = append(names, fmt.Sprintf("water700%02d.jpg", i))
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(root, name), encoded.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	w := worldstate.New()
	w.GND = &res.GND{Width: 1, Height: 1, Textures: []string{"ground.png", "ground.png", "missing.png"}, Cells: []res.GNDCell{{Top: 0}}, Lightmaps: []res.GNDLightmap{{}}}
	w.RSW = &res.RSW{Water: res.RSWWater{Level: 1, Type: 700}, Models: []res.RSWModel{{Filename: "test.rsm"}, {Filename: "test.rsm"}}}
	w.RSM = map[string]*res.RSM{"test.rsm": {Textures: []string{"ground.png", "model.png", "missing.png"}}}
	return client.Context{Resources: &res.Manager{Root: root}, World: w}
}
