package maps

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func mapTextureFixture(t *testing.T) (*System, client.Context) {
	t.Helper()
	return &System{Textures: make(map[string]*render.Image), TextureMiss: make(map[string]struct{})}, testutil.MapTextureContext(t)
}

func TestMapTexturesPreloadAllSourcesAndReuseDrawCaches(t *testing.T) {
	m, ctx := mapTextureFixture(t)
	for _, load := range m.TextureLoadTasks(ctx) {
		load()
	}
	if len(m.Images.Uploads) != 35 || len(m.Textures) != 34 || len(m.TextureMiss) != 1 {
		t.Fatalf("uploads=%d textures=%d missing=%d; want 2 shared/model + 32 water + lightmap", len(m.Images.Uploads), len(m.Textures), len(m.TextureMiss))
	}
	for _, name := range []string{"ground.png", "model.png"} {
		if img := m.Textures[name]; img.ByteSize() != 12 {
			t.Fatalf("%s: missing 1x1 mip below the 2x1 base", name)
		}
	}
	// Even if the file becomes unreadable, normal rendering uses preparation.
	if err := os.WriteFile(filepath.Join(ctx.Resources.Root, "ground.png"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	texture := m.groundTexture(ctx.Resources, "ground.png")
	if texture != m.Images.Uploads[0] || texture.RGBA().RGBAAt(0, 0) != (color.RGBA{}) || texture.RGBA().RGBAAt(1, 0) != (color.RGBA{G: 255, A: 255}) {
		t.Fatal("normal loader did not reuse preloaded pixels or transparency changed")
	}
	for frame := 0; frame < 32; frame++ {
		if img := m.waterTexture(ctx.Resources, 700, frame); img == nil || img.ByteSize() != 8 {
			t.Fatalf("water frame %d missing or mipmapped", frame)
		}
	}
	if atlas := m.gndMeshCache.lightmapAtlas.image; atlas == nil || atlas != m.Images.Uploads[1] || atlas.ByteSize() != len(atlas.RGBA().Pix) {
		t.Fatal("lightmap preparation was not retained")
	}
}

func TestMapTexturesSkipDryMapsAndHeadless(t *testing.T) {
	for _, headless := range []bool{false, true} {
		m, ctx := mapTextureFixture(t)
		ctx.Config.Headless = headless
		ctx.World.RSW.Water.Level = -1
		for _, load := range m.TextureLoadTasks(ctx) {
			load()
		}
		want := 3
		if headless {
			want = 0
		}
		if len(m.Images.Uploads) != want {
			t.Fatalf("headless=%t: prepared %d textures, want %d", headless, len(m.Images.Uploads), want)
		}
		// A dry map must not start lazy water loads during subsequent draws.
		// A nil resource manager makes any attempted file read fail this test.
		m.DrawGNDWater(render.NewFrame(100, 100), nil, ctx.World.GND, ctx.World.RSW, scene.Projection{}, time.Now(), scene.SceneFog{})
	}
}

type mapTextureRuntime struct {
	client.RuntimeSettings
	upscale bool
}

func (r *mapTextureRuntime) TextureUpscaling() bool { return r.upscale }

func TestMapTextureUpscalingIsCachedAndAppliesOnNextMap(t *testing.T) {
	m, ctx := mapTextureFixture(t)
	runtime := &mapTextureRuntime{upscale: true}
	ctx.Runtime = runtime
	for _, load := range m.TextureLoadTasks(ctx) {
		load()
	}
	for _, name := range []string{"ground.png", "model.png"} {
		img := m.Textures[name]
		if img.Bounds() != image.Rect(0, 0, 4, 2) || img.ByteSize() != 44 {
			t.Fatalf("%s was not upscaled with mipmaps", name)
		}
		if m.groundTexture(ctx.Resources, name) != img {
			t.Fatal("draw-time access did not reuse the upscaled texture")
		}
	}
	if m.waterTexture(ctx.Resources, 700, 0).ByteSize() != 8 {
		t.Fatal("upscaling changed a water texture")
	}
	if atlas := m.gndMeshCache.lightmapAtlas.image; atlas.ByteSize() != len(atlas.RGBA().Pix) {
		t.Fatal("upscaling changed the lightmap")
	}
	runtime.upscale = false
	delete(m.Textures, "model.png")
	if m.groundTexture(ctx.Resources, "model.png").Bounds().Dx() != 4 {
		t.Fatal("late model load ignored the current map's captured upscale setting")
	}
	m.Images.Release()
	m.Textures = make(map[string]*render.Image)
	ctx.Config.Render.TextureUpscaling = true // The runtime setting must take precedence.
	for _, load := range m.TextureLoadTasks(ctx) {
		load()
	}
	if m.Textures["ground.png"].Bounds().Dx() != 2 {
		t.Fatal("next map did not pick up the disabled runtime setting")
	}
	m.Images.Release()
}

func TestLoadMapModelsHasNo128ModelLimitAndCachesFailures(t *testing.T) {
	root := t.TempDir()
	// Valid RSM 1.5 with no nodes or textures.
	data := make([]byte, 83)
	copy(data, []byte{'G', 'R', 'S', 'M', 1, 5})
	rsw := &res.RSW{}
	for i := 0; i < 130; i++ {
		name := fmt.Sprintf("model%d.rsm", i)
		if err := os.WriteFile(filepath.Join(root, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
		rsw.Models = append(rsw.Models, res.RSWModel{Filename: name})
	}
	rsw.Models = append(rsw.Models, res.RSWModel{Filename: "missing.rsm"}, res.RSWModel{Filename: "missing.rsm"})
	w := worldstate.New()
	w.RSW = rsw
	for _, load := range ModelLoadTasks(client.Context{Resources: &res.Manager{Root: root}, World: w}) {
		load()
	}
	models, failures := w.RSM, w.RSMFail
	if len(models) != 131 || models["model129.rsm"] == nil || failures != 1 {
		t.Fatalf("models=%d failures=%d; late model=%v", len(models), failures, models["model129.rsm"])
	}
}
