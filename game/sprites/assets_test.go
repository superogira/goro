package sprites

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestSharedSpriteResourcesKeepViewStateAndPalettesIndependent(t *testing.T) {
	root := t.TempDir()
	act := make([]byte, 16)
	copy(act, []byte{'A', 'C', 1, 1})
	spr := append([]byte{'S', 'P', 1, 1, 1, 0, 2, 0, 1, 0, 1, 0}, make([]byte, 1024)...)
	spr[12+4] = 255 // Default palette: red.
	pal := make([]byte, 1024)
	pal[5] = 255 // Override palette: green.
	for name, data := range map[string][]byte{"body.act": act, "body.spr": spr, "green.pal": pal} {
		if err := os.WriteFile(filepath.Join(root, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	manager := &res.Manager{Root: root}
	load := func(palettes []string) *scene.SpriteView {
		t.Helper()
		view, status := scene.LoadSpriteView(manager, []string{"body.act"}, []string{"body.spr"}, palettes, "test")
		if view == nil {
			t.Fatal(status)
		}
		return view
	}
	first := load(nil)
	first.Started = time.Unix(100, 0)
	first.Billboards[scene.SpriteBillboardKey{}] = &scene.Billboard{}
	first.Images[scene.SpriteFrameKey{Index: 100}] = render.NewImage(1, 1)
	second := load([]string{"green.pal"})
	if first == second || first.Spr != second.Spr || first.Act != second.Act {
		t.Fatal("views did not share just their parsed resources")
	}
	if first.Started == second.Started || len(second.Billboards) != 0 || len(second.Images) != 0 {
		t.Fatal("animation or rendering state leaked between views")
	}
	for view, want := range map[*scene.SpriteView]color.RGBA{
		first: {R: 255, A: 255}, second: {G: 255, A: 255},
	} {
		img, ok := scene.SpriteViewImage(view, 0, res.SPRFramePalette)
		if !ok || img.RGBA().RGBAAt(0, 0) != want || img.RGBA().RGBAAt(1, 0) != (color.RGBA{}) {
			t.Fatalf("wrong color or transparency for palette %q", view.PaletteSource)
		}
	}
	// A fresh view, as created after a map change, still gets the same CPU
	// resources and default palette, without retaining the old view's images.
	third := load(nil)
	if third.Spr != first.Spr || third.Act != first.Act || third.Palette != nil || len(third.Images) != 0 {
		t.Fatal("fresh view did not reuse the unmodified resources")
	}
	if fourth := load([]string{"green.pal"}); fourth.Palette != second.Palette {
		t.Fatal("palette override not shared")
	}
}

func TestHumanoidViewsShareResourcesRealData(t *testing.T) {
	manager := testutil.RealDataManager(t)
	first, status := LoadHumanoid(manager, Appearance{Job: 4, Head: 1, Sex: 1}, "first")
	if first == nil {
		t.Fatal(status)
	}
	second, status := LoadHumanoid(manager, Appearance{Job: 4, Head: 2, Sex: 1}, "second")
	if second == nil {
		t.Fatal(status)
	}
	if first.Body == second.Body || first.Body.Spr != second.Body.Spr || first.Body.Act != second.Body.Act || first.imf != second.imf {
		t.Fatal("different appearances did not share their common resources")
	}
	if first.head == nil || second.head == nil || first.head.Spr == second.head.Spr {
		t.Fatal("different heads lost their own resources")
	}
	for _, view := range []*Humanoid{first, second} {
		if _, ok := HumanoidBillboard(view, State{ActionFamily: scene.SpriteActionIdle, Direction: 0}, time.Now()); !ok {
			t.Fatal("shared resources could not render")
		}
	}
}

func TestMonsterViewsShareUpgradedResourcesRealData(t *testing.T) {
	manager := testutil.RealDataManager(t)
	first, status := LoadNonPC(manager, 1002, "poring")
	if first == nil || len(first.Act.Actions) < 40 {
		t.Fatalf("richer monster resource missing: %s", status)
	}
	second, status := LoadNonPC(manager, 1002, "poring again")
	if second == nil || first == second || first.Act != second.Act || first.Spr != second.Spr {
		t.Fatalf("upgraded monster resources not shared: %s", status)
	}
}
