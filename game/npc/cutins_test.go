package npc

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"golang.org/x/image/bmp"
)

func TestApplyNPCCutinLoadsAndClearsIllustration(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "data", "texture", "유저인터페이스", "illust", "guide.bmp")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	source := image.NewNRGBA(image.Rect(0, 0, 18, 24))
	source.Set(4, 5, color.NRGBA{R: 80, G: 120, B: 160, A: 255})
	if err := bmp.Encode(file, source); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	mode := newTestSystem()
	ctx := client.Context{Resources: &res.Manager{Root: root}}
	if err := mode.ApplyCutin(ctx, network.NPCCutin{Image: "guide", Position: network.NPCCutinRight}); err != nil {
		t.Fatal(err)
	}
	if !mode.bindings.Windows.Cutin.Visible() {
		t.Fatal("loaded cut-in is not visible")
	}

	if err := mode.ApplyCutin(ctx, network.NPCCutin{Position: network.NPCCutinClear}); err != nil {
		t.Fatal(err)
	}
	if mode.bindings.Windows.Cutin.Visible() {
		t.Fatal("clear packet left cut-in visible")
	}
}
