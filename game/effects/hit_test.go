package effects

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func TestHitRingGeometryAndCapturedFacing(t *testing.T) {
	spec, _ := Lookup(Hit1)
	ring := spec.Components[1]
	starts := time.Unix(100, 0)
	for _, tt := range []struct {
		dir  int
		axis scene.Point3
	}{
		{0, scene.Point3{Z: 1}},
		{2, scene.Point3{X: -1}},
		{4, scene.Point3{Z: -1}},
		{6, scene.Point3{X: 1}},
	} {
		world := worldstate.New()
		world.Actors[42] = worldstate.Actor{ID: 42, Dir: tt.dir}
		ctx := client.Context{World: world, Resources: &res.Manager{}}
		mode := &System{textures: map[string]*render.Image{"__effect_ring_blue": render.NewImage(64, 64)}}
		if !mode.AddAt(ctx, Hit1, 42, starts) {
			t.Fatal("failed to add regular hit")
		}
		mode.Draw(render.NewFrame(800, 600), ctx, scene.Projection{}, starts, false)
		// Turning after the hit must not rotate the cylinder mid-flight.
		world.Actors[42] = worldstate.Actor{ID: 42, Dir: (tt.dir + 2) % 8}
		frame := render.NewFrame(800, 600)
		mode.DrawFuncEffect(frame, ctx, scene.Projection{}, mode.Active[0], ring, 1, 10, 20, 3, 0, starts)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if commands.Len() != 1 {
			t.Fatalf("direction %d: got %d draws, want a cylinder", tt.dir, commands.Len())
		}
		command := commands.Index(0)
		vertices := command.FieldByName("Vertices")
		if vertices.Len() != 22 || command.FieldByName("Indices").Len() != 60 {
			t.Fatal("expected ten cylinder faces")
		}
		point := func(i int) scene.Point3 {
			v := vertices.Index(i)
			return scene.Point3{X: v.FieldByName("X").Float(), Y: v.FieldByName("Y").Float(), Z: v.FieldByName("Z").Float()}
		}
		var bottom, top scene.Point3
		for i := 0; i < 10; i++ {
			bottom = scene.Add3(bottom, scene.Mul3(point(2*i), 0.1))
			top = scene.Add3(top, scene.Mul3(point(2*i+1), 0.1))
		}
		// The origin, like the facing, stays at the captured impact position.
		wantBottom := scene.Add3(scene.Point3{X: .5, Y: 2.07, Z: .5}, scene.Mul3(tt.axis, 0.133))
		wantTop := scene.Add3(wantBottom, scene.Mul3(tt.axis, 0.7))
		if !testutil.ModelPointNear(bottom, wantBottom, 1e-5) || !testutil.ModelPointNear(top, wantTop, 1e-5) {
			t.Fatalf("direction %d: centers = %+v/%+v, want %+v/%+v", tt.dir, bottom, top, wantBottom, wantTop)
		}
		radius := scene.Sub3(point(0), bottom)
		if !testutil.ModelPointNear(scene.Sub3(point(1), top), scene.Mul3(radius, 2), 1e-5) || math.Abs(scene.Dot3(radius, radius)-1) > 1e-5 {
			t.Fatal("expected aligned inner/outer radii of 1/2 cells")
		}
		if got := vertices.Index(20).FieldByName("SrcX").Float(); got != 160 {
			t.Fatalf("final U = %g, want 2.5 texture repeats", got)
		}
		options := command.FieldByName("Options")
		if render.Blend(options.FieldByName("Blend").Int()) != render.BlendSourceOver || !options.FieldByName("DepthTest").Bool() || options.FieldByName("DepthWrite").Bool() {
			t.Fatal("ring must alpha blend, test world depth, and not write depth")
		}
	}
}

func TestHitRingDrawsWithOldROData(t *testing.T) {
	manager := testutil.RealDataManager(t)
	world := worldstate.New()
	world.Actors[42] = worldstate.Actor{ID: 42, X: 10, Y: 20}
	ctx := client.Context{World: world, Resources: manager}
	mode := &System{}
	starts := time.Unix(100, 0)
	mode.AddAt(ctx, Hit1, 42, starts)
	projection := testutil.Projection(800, 600, 10.5, 20.5, 0)
	for _, elapsed := range []time.Duration{0, 100 * time.Millisecond, 200 * time.Millisecond} {
		frame := render.NewFrame(800, 600)
		mode.Draw(frame, ctx, projection, starts.Add(elapsed), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		rings := 0
		for i := 0; i < commands.Len(); i++ {
			if commands.Index(i).FieldByName("Vertices").Len() == 22 {
				rings++
			}
		}
		if want := elapsed < 150*time.Millisecond; (rings == 1) != want {
			t.Fatalf("elapsed %s: rings = %d, want visible=%t", elapsed, rings, want)
		}
	}
}

func TestHitRingOriginalAnimation(t *testing.T) {
	spec, _ := Lookup(Hit1)
	ring := spec.Components[1]
	for _, tt := range []struct {
		frame           int
		distance, alpha float64
	}{
		{-1, 0, 0},
		{0, 0.133, 254.0 / 255},
		{4, 0.595, 254.0 / 255},
		{5, 0.693, 203.2 / 255},
		{8, 0.945, 50.8 / 255},
		{9, 1.015, 0},
	} {
		distance, alpha := hitRingAnimation(ring, time.Duration(tt.frame)*ring.FrameDelay)
		if math.Abs(distance-tt.distance) > 1e-6 || math.Abs(alpha-tt.alpha) > 1e-6 {
			t.Errorf("frame %d: distance/alpha = %g/%g, want %g/%g", tt.frame, distance, alpha, tt.distance, tt.alpha)
		}
	}
	if _, alpha := hitRingAnimation(ring, ring.Duration); alpha != 0 {
		t.Fatal("ring should expire before the 300ms sparks")
	}
}

func TestHitRingCapturesFacingAtImpact(t *testing.T) {
	world := worldstate.New()
	world.Actors[42] = worldstate.Actor{ID: 42, Dir: 0}
	ctx := client.Context{World: world}
	mode := &System{}
	starts := time.Unix(100, 0)
	mode.AddAt(ctx, Hit1, 42, starts)
	draw := func(at time.Time) {
		mode.Draw(render.NewFrame(800, 600), ctx, scene.Projection{}, at, false)
	}
	draw(starts.Add(-time.Millisecond))
	if mode.Active[0].HasActorTransform {
		t.Fatal("facing captured before the scheduled impact")
	}
	world.Actors[42] = worldstate.Actor{ID: 42, Dir: 2, X: 12, Y: 14}
	draw(starts)
	if effect := mode.Active[0]; !effect.HasActorTransform || effect.ActorDirection != 2 {
		t.Fatalf("impact facing = %d, want current target direction 2", effect.ActorDirection)
	}
	wantOrigin := scene.Point3{X: 12.5, Y: .07, Z: 14.5}
	if mode.Active[0].ActorOrigin != wantOrigin {
		t.Fatal("origin captured before the scheduled impact")
	}
	world.Actors[42] = worldstate.Actor{ID: 42, Dir: 4, X: 30, Y: 40}
	draw(starts.Add(50 * time.Millisecond))
	if effect := mode.Active[0]; effect.ActorDirection != 2 || effect.ActorOrigin != wantOrigin {
		t.Fatal("ring moved or turned with the actor after impact")
	}
}
