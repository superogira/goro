package effects

import (
	"math"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestSwordmanRenderingAndMovement(t *testing.T) {
	ctx := testutil.SwordmanContext()
	starts := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 10, 20, 0)
	for _, id := range []int{BashBegin, BashHit, Endure, MagnumBreak} {
		spec, _ := Lookup(id)
		component := spec.Components[0]
		m := &System{textures: map[string]*render.Image{}}
		for _, path := range append([]string{component.TextureFile}, component.TextureFiles...) {
			m.textures["__effectfile_"+path+"|blackkey=true"] = render.NewImage(8, 8)
		}
		ctx.World.Player.X = 10
		m.AddAt(ctx, id, 2000000, starts)
		ctx.World.Player.X = 12
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, starts.Add(5*db.EffectFrameDuration), false)
		value := reflect.ValueOf(frame).Elem()
		commands, overlays := value.FieldByName("worldCommands"), value.FieldByName("commands")
		if commands.Len()+overlays.Len() == 0 {
			t.Fatalf("effect %d rendered nothing", id)
		}
		if value.FieldByName("worldBillboards").Len() != 0 {
			t.Fatalf("effect %d unexpectedly queued a world billboard", id)
		}
		if (id == MagnumBreak && overlays.Len() != 0) || (id != MagnumBreak && commands.Len() != 0) {
			t.Fatalf("effect %d uses the wrong render pass", id)
		}
		for _, draws := range []reflect.Value{commands, overlays} {
			for i := 0; i < draws.Len(); i++ {
				options := draws.Index(i).FieldByName("Options")
				if options.FieldByName("Blend").Int() != int64(render.BlendSourceOver) || options.FieldByName("DepthWrite").Bool() {
					t.Fatalf("effect %d changed native blending or writes depth", id)
				}
				if options.FieldByName("DepthTest").Bool() != (id == MagnumBreak && i == 0) {
					t.Fatalf("effect %d wrong depth behavior", id)
				}
			}
		}
		if id == MagnumBreak {
			if commands.Len() != 3 {
				t.Fatal("Magnum Break must draw a sphere and two rings")
			}
			verts := commands.Index(1).FieldByName("Vertices")
			minX, maxX := math.Inf(1), math.Inf(-1)
			for i := 0; i < verts.Len(); i++ {
				x := verts.Index(i).FieldByName("X").Float()
				minX, maxX = math.Min(minX, x), math.Max(maxX, x)
			}
			if math.Abs((minX+maxX)/2-10.5) > .001 {
				t.Fatal("Magnum Break followed the caster instead of staying at the cast position")
			}
		} else if id == BashBegin {
			for _, zoom := range []float64{75, 125, 200} {
				projection := scene.NewProjection(800, 600, 10, 20, 0, 45, 50, zoom)
				frame := render.NewFrame(800, 600)
				m.Draw(frame, ctx, projection, starts.Add(5*db.EffectFrameDuration), false)
				verts := reflect.ValueOf(frame).Elem().FieldByName("commands").Index(0).FieldByName("Vertices")
				center := verts.Index(0)
				cx, cy := center.FieldByName("DstX").Float(), center.FieldByName("DstY").Float()
				want := projection.Project(12.5, 20.5, .07)
				if math.Abs(cx-float64(want.X)) > .001 || math.Abs(cy-float64(want.Y-40)) > .001 {
					t.Fatal("Bash must follow the caster, forty pixels above its feet")
				}
				for i := 0; i < verts.Len(); i++ {
					vertex := verts.Index(i)
					radius := math.Hypot(vertex.FieldByName("DstX").Float()-cx, vertex.FieldByName("DstY").Float()-cy)
					if i%3 != 0 && math.Abs(radius-100) > .001 {
						t.Fatal("Bash's screen-space radius changed with camera zoom")
					}
				}
			}
		}
		expired := render.NewFrame(800, 600)
		m.Draw(expired, ctx, projection, starts.Add(spec.Duration+time.Second), false)
		value = reflect.ValueOf(expired).Elem()
		if value.FieldByName("commands").Len()+value.FieldByName("worldCommands").Len()+value.FieldByName("worldBillboards").Len() != 0 {
			t.Fatalf("effect %d rendered after its lifetime", id)
		}
	}
}

func TestSwordmanNativeTimings(t *testing.T) {
	for _, tc := range []struct {
		frame, life, rise, fade int
		peak                    float64
		want                    uint8
	}{
		{0, 40, 6, 30, 170, 28}, {5, 40, 6, 30, 170, 170}, {30, 40, 6, 30, 170, 153}, {39, 40, 6, 30, 170, 0},
		{0, 80, 20, 54, 254, 12}, {19, 80, 20, 54, 254, 254}, {79, 80, 20, 54, 254, 0},
		{0, 30, 15, 15, 254, 16}, {14, 30, 15, 15, 254, 254}, {29, 30, 15, 15, 254, 0},
	} {
		if got := effectParticleAlpha(tc.frame, tc.life, tc.rise, tc.fade, tc.peak); got != tc.want {
			t.Fatalf("phase %+v: alpha %d", tc, got)
		}
	}
	effect := Instance{Starts: time.Unix(100, 0), ActorID: 300}
	for i := 0; i < 8; i++ {
		first, later := bashHitParticle(effect, i, 0), bashHitParticle(effect, i, 7)
		if first.Alpha != 31 || later.Alpha != 254 || later.halfWidth >= first.halfWidth || later.halfHeight <= first.halfHeight {
			t.Fatalf("Bash impact must brighten, narrow and stretch: %+v -> %+v", first, later)
		}
		if p := bashHitParticle(effect, i, 29); p.Alpha != 0 {
			t.Fatal("Bash impact outlived its native maximum")
		}
	}
	for birth := 0; birth < 50; birth++ {
		first, later := endureStreak(effect, birth, birth), endureStreak(effect, birth, birth+19)
		if birth <= 40 {
			if first.Alpha != 12 || later.Alpha != 254 || math.Hypot(later.x, later.Y+20) >= math.Hypot(first.x, first.Y+20) {
				t.Fatalf("Endure streak must brighten and travel inward: %+v -> %+v", first, later)
			}
		} else if first.Alpha != 0 || later.Alpha != 0 {
			t.Fatal("Endure emitted a streak after the native parent fade cutoff")
		}
		if endureStreak(effect, birth, birth-1).Alpha != 0 || endureStreak(effect, birth, birth+39).Alpha != 0 {
			t.Fatal("Endure streak visible outside its lifetime")
		}
	}
}

func TestMagnumSphereGeometry(t *testing.T) {
	texture := render.NewImage(16, 16)
	origin := scene.Point3{X: 10, Y: 2, Z: 20}
	vertices, indices := effectSphereMesh(texture, origin, magnumRadius(1.15, 14), 45, 254)
	if len(vertices) != 200 || len(indices) != 300 {
		t.Fatal("Magnum Break must use the native ten by five sphere mesh")
	}
	// Fifteen updates: v += -1.15/60, r += v; five native units per cell.
	const radius = 2.99
	below, above := false, false
	for _, vertex := range vertices {
		p := scene.Point3{X: float64(vertex.X), Y: float64(vertex.Y), Z: float64(vertex.Z)}
		offset := scene.Sub3(p, origin)
		if math.Abs(math.Sqrt(scene.Dot3(offset, offset))-radius) > .00001 {
			t.Fatalf("sphere radius at peak opacity = %+v", offset)
		}
		below, above = below || p.Y < origin.Y, above || p.Y > origin.Y
	}
	if !below || !above {
		t.Fatal("Magnum Break must be a sphere intersecting the floor, not a cylinder")
	}
	for _, index := range indices {
		if int(index) >= len(vertices) {
			t.Fatal("out-of-bounds sphere index")
		}
	}
}

func TestSwordmanRealAssets(t *testing.T) {
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		t.Skip("set GORO_DATA_DIR to check real effect assets")
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, archive := range manager.Archives {
		t.Cleanup(func() { _ = archive.Close() })
	}
	m := &System{}
	for _, id := range []int{BashBegin, BashHit, Endure, MagnumBreak, Provoke} {
		spec, _ := Lookup(id)
		for _, c := range spec.Components {
			for _, path := range append([]string{c.TextureFile}, c.TextureFiles...) {
				if path != "" && m.EffectFileTexture(manager, path) == nil {
					t.Errorf("effect %d missing texture %s", id, path)
				}
			}
			if c.StrFile != "" && m.LoadWorldEffectSTR(manager, c.StrFile, c.TexturePath) == nil {
				t.Errorf("effect %d missing STR %s", id, c.StrFile)
			}
		}
		for _, sound := range spec.Sfx {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Error(err)
			}
		}
	}
}
