package effects

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func TestOriginalHitCylinderDimensionsAndMotion(t *testing.T) {
	starts := time.Unix(100, 0)
	for _, tt := range []struct {
		id, index                      int
		inner, outer, height, distance float64
	}{
		{Hit3, 0, .3, .3, .14, .7 / 5 * (1 - 1.0/30)},
		{Hit3, 1, .3, .8, .09, .7 / 5 * (1 - 1.0/30)},
		{Hit4, 0, .1, .8, .08, .7 / 5 * (1 - 1.0/30)},
		{SonicBlowHit, 0, .8, 1.4, .7, .4 + .4/5*(1-1.0/24)},
	} {
		spec, _ := Lookup(tt.id)
		component := spec.Components[tt.index]
		effect := Instance{EffectID: tt.id, Starts: starts, ActorDirection: 0}
		mode := System{textures: map[string]*render.Image{"__effect_" + component.TextureName: render.NewImage(64, 64)}}
		frame := render.NewFrame(800, 600)
		mode.DrawFuncEffect(frame, client.Context{Resources: &res.Manager{}}, scene.Projection{}, effect, component, tt.index, 10, 20, 3, 0, starts)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if commands.Len() != 1 {
			t.Fatalf("effect %d component %d: no cylinder", tt.id, tt.index)
		}
		vertices := commands.Index(0).FieldByName("Vertices")
		var centers [2]scene.Point3
		for i := 0; i < 20; i++ {
			v := vertices.Index(i)
			centers[i%2] = scene.Add3(centers[i%2], scene.Mul3(scene.Point3{X: v.FieldByName("X").Float(), Y: v.FieldByName("Y").Float(), Z: v.FieldByName("Z").Float()}, .1))
		}
		if !testutil.ModelPointNear(scene.Sub3(centers[1], centers[0]), scene.Point3{Z: tt.height}, 1e-5) || math.Abs(centers[0].Z-20-tt.distance) > 1e-5 {
			t.Fatalf("effect %d component %d: cylinder centres = %+v", tt.id, tt.index, centers)
		}
		for side, wantRadius := range []float64{tt.inner, tt.outer} {
			v := vertices.Index(side)
			p := scene.Point3{X: v.FieldByName("X").Float(), Y: v.FieldByName("Y").Float(), Z: v.FieldByName("Z").Float()}
			d := scene.Sub3(p, centers[side])
			if math.Abs(math.Sqrt(scene.Dot3(d, d))-wantRadius) > 1e-5 {
				t.Fatalf("effect %d: wrong radius", tt.id)
			}
		}
		if tt.id == SonicBlowHit {
			if centers[0].X < 10-1e-5 || centers[0].X > 10.4+1e-5 || centers[0].Y < 3.2-1e-5 || centers[0].Y > 5.2+1e-5 {
				t.Fatalf("sonic impact offset = %+v", centers[0])
			}
		} else if math.Abs(centers[0].Y-5) > 1e-5 {
			t.Fatalf("hit altitude = %g", centers[0].Y)
		}
		_, alpha := hitRingAnimation(component, 9*db.EffectFrameDuration)
		if tt.id != SonicBlowHit && math.Abs(alpha-254.0/255) > 1e-9 {
			t.Fatal("legacy hit fades before tick ten")
		}
	}
}

func TestOriginalEffectsRenderWithOldROAssets(t *testing.T) {
	manager := testutil.RealDataManager(t)
	world := worldstate.New()
	ctx := client.Context{World: world, Resources: manager}
	starts := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 10.5, 20.5, 0)
	for _, id := range []int{SonicBlowHit, Hit3, Hit4, Blessing} {
		world.Actors[42] = worldstate.Actor{ID: 42, X: 10, Y: 20}
		mode := System{}
		if !mode.AddAt(ctx, id, 42, starts) {
			t.Fatalf("effect %d failed to spawn", id)
		}
		frame := render.NewFrame(800, 600)
		mode.Draw(frame, ctx, projection, starts.Add(2*db.EffectFrameDuration), false)
		if len(mode.textureMiss) > 0 || len(mode.EffectViewMiss) > 0 {
			t.Fatalf("effect %d missing resources: %v %v", id, mode.textureMiss, mode.EffectViewMiss)
		}
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if commands.Len() == 0 {
			t.Fatalf("effect %d drew nothing", id)
		}
		if id == Hit3 || id == Hit4 {
			count := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len()
			want := 24
			if id == Hit4 {
				want = 15
			}
			if count != want {
				t.Fatalf("effect %d: got %d particle sprites, want %d", id, count, want)
			}
		}
		if id != Blessing {
			world.Actors[42] = worldstate.Actor{ID: 42, X: 30, Y: 40, Dir: 4}
			moved := render.NewFrame(800, 600)
			mode.Draw(moved, ctx, projection, starts.Add(2*db.EffectFrameDuration), false)
			if !reflect.DeepEqual(frame, moved) {
				t.Fatalf("effect %d moved with the target instead of staying at impact", id)
			}
		}
	}
}

func TestHitParticleTrailUsesEarlierPositionsAndCurrentFade(t *testing.T) {
	p := hitParticle{axis: scene.Point3{Z: 1}, radius: 1, speed: .2, size: 1.2, frames: 10}
	// Three updates: distances .19, .18, .17; history trails by one/two ticks.
	for trail, distance := range []float64{1.54, 1.37, 1.19} {
		position, size, alpha := p.segment(2, trail, 2)
		if !testutil.ModelPointNear(position, scene.Point3{Y: 2, Z: distance}, 1e-9) {
			t.Fatalf("trail %d position = %+v", trail, position)
		}
		if math.Abs(size-.84*(1-float64(trail)/6)) > 1e-9 || math.Abs(alpha-(254.0/255)*.7*(1-float64(trail)/3)) > 1e-9 {
			t.Fatalf("trail %d size/alpha = %g/%g", trail, size, alpha)
		}
	}
	if pos, _, _ := p.segment(0, 2, 2); pos != (scene.Point3{}) {
		t.Fatal("initial trail history must start at actor anchor")
	}
	if _, size, alpha := p.segment(9, 0, 2); size != 0 || alpha != 0 {
		t.Fatal("particle did not shrink/fade completely")
	}
	for _, id := range []int{Hit3, Hit4} {
		spec, _ := Lookup(id)
		component := spec.Components[len(spec.Components)-1]
		want := 8
		if id == Hit4 {
			want = 5
		}
		if component.Duplicate != want {
			t.Fatalf("effect %d: want %d trails", id, want)
		}
		for i := 0; i < 32; i++ {
			p := newHitParticle(Instance{EffectID: id, Starts: time.Unix(100, 0)}, component, i)
			if p.frames < 6 || p.frames > 29 || p.size < .6 || p.size >= 1.6 || math.Abs(scene.Dot3(p.axis, p.axis)-1) > 1e-9 {
				t.Fatalf("invalid randomized particle: %+v", p)
			}
		}
	}
}
