package effects

import (
	"math"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestAquaBenedictaPlaysThroughLastFrame(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	spec, _ := Lookup(Aqua)
	sprite := testutil.EffectRenderTestSprite()
	sprite.Act.Actions[0] = res.ACTAction{DelayMS: 25, Animations: make([]res.ACTAnimation, 42)}
	lastFrame := scene.SpriteBillboardKey{Motion: 41}
	sprite.Billboards[lastFrame] = sprite.Billboards[scene.SpriteBillboardKey{}]
	m := &System{EffectViews: map[string]*scene.SpriteView{spec.Components[0].SpriteFile: sprite}}
	start := time.Unix(100, 0)
	m.AddAt(ctx, Aqua, 300, start)
	end := start.Add(84 * db.EffectFrameDuration)
	if len(m.Active) != 1 || !m.Active[0].Expires.Equal(end) {
		t.Fatalf("Aqua must last 42 frames of two native ticks: %+v", m.Active)
	}
	for _, at := range []time.Time{end.Add(-time.Millisecond), end} {
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, testutil.Projection(800, 600, 30.5, 20.5, 0), at, false)
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len()
		if (draws == 1) != at.Before(end) {
			t.Fatalf("Aqua at %s: got %d draws, must show its last frame until expiry", at.Sub(start), draws)
		}
	}
}

func TestAcolyteRealAssets(t *testing.T) {
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		t.Skip("set GORO_DATA_DIR to validate original Acolyte assets")
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, archive := range manager.Archives {
		t.Cleanup(func() { _ = archive.Close() })
	}
	ctx := testutil.KnightEffectContext()
	ctx.Resources = manager
	start := time.Unix(100, 0)
	for _, id := range []int{
		Exit, Heal, Heal2, Heal4, HealOffensive,
		IncAgility, DecAgility, Aqua, Signum, Angelus,
		Blessing, Cure, HolyLight, Ruwach, Pneuma,
		Teleportation, ReadyPortal, Portal,
	} {
		m := &System{}
		spec, _ := Lookup(id)
		for _, sound := range spec.Sfx {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Fatalf("effect %d: %v", id, err)
			}
		}
		m.AddAt(ctx, id, 300, start)
		drew := false
		for _, tick := range []int{0, 15, 30, 60, 90} {
			frame := render.NewFrame(800, 600)
			projection := testutil.Projection(800, 600, 30.5, 20.5, 0)
			m.Draw(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			value := reflect.ValueOf(frame).Elem()
			drew = drew || value.FieldByName("worldCommands").Len()+value.FieldByName("commands").Len()+value.FieldByName("worldBillboards").Len() > 0
		}
		if !drew {
			t.Fatalf("effect %d drew nothing", id)
		}
	}
}

func TestBlessingCircleIsFlatAndUsesOriginalFade(t *testing.T) {
	spec, _ := Lookup(Blessing)
	component := spec.Components[3]
	starts := time.Unix(100, 0)
	mode := System{textures: map[string]*render.Image{"__effect_alpha_down": render.NewImage(64, 64)}}
	for _, tt := range []struct {
		frame int
		alpha float64
	}{{-1, 0}, {0, 100.0 / 255 / 30}, {29, 100.0 / 255}, {119, 100.0 / 255}, {120, 100.0 / 255 * 29 / 30}, {149, 0}, {150, 0}} {
		if got := blessingCircleAlpha(component, time.Duration(tt.frame)*db.EffectFrameDuration); math.Abs(got-tt.alpha) > 1e-9 {
			t.Fatalf("frame %d alpha = %g want %g", tt.frame, got, tt.alpha)
		}
	}
	frame := render.NewFrame(800, 600)
	mode.drawBlessingCircleEffect(frame, client.Context{Resources: &res.Manager{}}, component, Instance{Starts: starts}, 10, 20, 3, starts.Add(40*db.EffectFrameDuration))
	commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
	if commands.Len() != 1 {
		t.Fatal("missing blessing circle")
	}
	cmd := commands.Index(0)
	vertices := cmd.FieldByName("Vertices")
	for i := 0; i < vertices.Len(); i++ {
		v := vertices.Index(i)
		x, y, z := v.FieldByName("X").Float(), v.FieldByName("Y").Float(), v.FieldByName("Z").Float()
		radius := math.Hypot(x-10, z-20)
		wantRadius := 2.0
		if i%3 == 0 {
			wantRadius = 0
		}
		if y != 3 || math.Abs(radius-wantRadius) > 1e-5 {
			t.Fatalf("vertex %d is not on the ground circle: %g,%g,%g", i, x, y, z)
		}
	}
	options := cmd.FieldByName("Options")
	if options.FieldByName("DepthTest").Bool() || options.FieldByName("DepthWrite").Bool() || render.Blend(options.FieldByName("Blend").Int()) != render.BlendSourceOver {
		t.Fatal("blessing must use RF_ALPHA|RF_NODEPTHCHECK")
	}
}

func TestAgilityNativeLabels(t *testing.T) {
	for _, decrease := range []bool{false, true} {
		label := AgilityLabel(decrease, 49, 100)
		wantY, wantHeight := -52.05, 20.0
		if decrease {
			wantY, wantHeight = -42.75, 10
		}
		if label.halfWidth != 40 || label.halfHeight != wantHeight || math.Abs(label.Y-wantY) > .0001 || label.Alpha != 200 {
			t.Fatalf("decrease=%t: native pixel dimensions, acceleration or fade changed: %+v", decrease, label)
		}
		if AgilityLabel(decrease, 0, 100).Alpha >= label.Alpha || AgilityLabel(decrease, 99, 100).Alpha != 0 {
			t.Fatal("label must fade at both ends")
		}
	}
}

func TestBlessingEffectSpecUsesRobrowserSpritesAndParticles(t *testing.T) {
	spec, ok := Lookup(Blessing)
	if !ok {
		t.Fatal("blessing effect spec missing")
	}
	if spec.Duration != 2500*time.Millisecond {
		t.Fatalf("duration = %s, want 2500ms", spec.Duration)
	}
	if len(spec.Sfx) != 1 || spec.Sfx[0] != "effect\\ef_blessing.wav" {
		t.Fatalf("sfx = %#v", spec.Sfx)
	}
	if len(spec.Components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.Components))
	}
	sprite := spec.Components[0]
	if sprite.Kind != ComponentSPR || sprite.SpriteFile != "축복" {
		t.Fatalf("sprite component = %+v", sprite)
	}
	if sprite.Duration != 1500*time.Millisecond || sprite.SpriteDelay != 30*time.Millisecond || !sprite.SpriteRepeat || !sprite.SpriteHead || sprite.SpriteYOffset != -120 || !sprite.WorldSizedSprite {
		t.Fatalf("sprite timing/placement = %+v", sprite)
	}

	particleCases := []struct {
		index       int
		delay       time.Duration
		posXRand    float64
		posYRand    float64
		sparkling   bool
		sparkNumber int
	}{
		{index: 1, delay: 300 * time.Millisecond, posXRand: 1.2, posYRand: 1, sparkling: true, sparkNumber: 2},
		{index: 2, delay: 400 * time.Millisecond, posXRand: 1.4, posYRand: 1.1},
	}
	for _, tc := range particleCases {
		component := spec.Components[tc.index]
		if component.Kind != Component3D || component.SpriteFile != "particle6" {
			t.Fatalf("particle %d resource = %+v", tc.index, component)
		}
		if component.Duration != 1200*time.Millisecond || component.Delay != tc.delay || component.DuplicateDelay != 0 || component.Duplicate != 6 {
			t.Fatalf("particle %d timing = %+v", tc.index, component)
		}
		if component.AlphaMax != 1 || !component.FadeIn || !component.FadeOut || component.Sparkling != tc.sparkling || component.SparkNumber != tc.sparkNumber {
			t.Fatalf("particle %d fade/sparkle = %+v", tc.index, component)
		}
		if component.PosXRand != tc.posXRand || component.PosYRand != tc.posYRand || component.PosZStartRand != 2 || component.PosZStartMiddle != 5.5 || component.PosZEndRand != 0.5 || component.PosZEndMiddle != 1 {
			t.Fatalf("particle %d position = %+v", tc.index, component)
		}
		if component.SizeStart != 50*scene.PixelRatio || component.SizeEnd != 50*scene.PixelRatio {
			t.Fatalf("particle %d size = %+v", tc.index, component)
		}
	}

	aura := spec.Components[3]
	if aura.FuncAdapter != effectFuncBlessingCircle || aura.TextureName != "alpha_down" || !aura.Overlay {
		t.Fatalf("blessing ground circle = %+v", aura)
	}
}
