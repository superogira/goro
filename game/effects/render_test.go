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

func TestEffectShadowFollowsTerrainAtMovingPosition(t *testing.T) {
	world := worldstate.New()
	world.GAT = &res.GAT{Width: 3, Height: 1, Cells: []res.GATCell{
		{}, {Heights: [4]float32{6, 6, 6, 6}}, {},
	}}
	ctx := client.Context{World: world, Resources: &res.Manager{}}
	mode := &System{EffectViews: map[string]*scene.SpriteView{"shadow": testutil.EffectRenderTestSprite()}}
	projection := testutil.Projection(800, 600, 1.5, .5, 0)
	starts := time.Unix(100, 0)
	for _, shadow := range []bool{false, true} {
		component := Component{
			SpriteFile: "shadow", ShadowTexture: shadow, Duration: time.Second,
			PosXEnd: 2, PosZ: 4, SizeStart: 1, SizeEnd: 1,
		}
		frame := render.NewFrame(800, 600)
		mode.draw3DEffect(frame, ctx, projection, Instance{Starts: starts}, component, 0, .5, .5, 10, starts.Add(500*time.Millisecond))
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if commands.Len() != 1 {
			t.Fatalf("shadow=%t: draws = %d, want 1", shadow, commands.Len())
		}
		center := commands.Index(0).FieldByName("Center")
		wantZ := 14.0
		if shadow {
			wantZ = 6 + scene.ActorShadowTerrainLift
		}
		if math.Abs(center.Index(0).Float()-1.5) > 1e-5 || math.Abs(center.Index(1).Float()-wantZ) > 1e-5 {
			t.Fatalf("shadow=%t: center=%v, want x=1.5 height=%g", shadow, center, wantZ)
		}
	}
}

func TestWorldEffectDrawReusesConvertedSpecs(t *testing.T) {
	ctx := client.Context{World: worldstate.New(), Resources: &res.Manager{}}
	now := time.Unix(100, 0)
	emotion := testutil.EffectRenderTestSprite()
	emotion.Billboards[scene.SpriteBillboardKey{AnchorY: -100}] = emotion.Billboards[scene.SpriteBillboardKey{}]
	mode := &System{
		EffectViews: map[string]*scene.SpriteView{"emotion": emotion}, Active: []Instance{{EffectID: Emotion, Starts: now}},
	}
	frame := render.NewFrame(800, 600)
	projection := testutil.Projection(800, 600, .5, .5, 0)
	allocations := testing.AllocsPerRun(10, func() {
		frame.BeginFrame()
		mode.Draw(frame, ctx, projection, now, true)
		mode.Draw(frame, ctx, projection, now, false)
	})
	if allocations != 0 {
		t.Fatalf("warmed effect draws allocate %g times per frame, want 0", allocations)
	}
	if got := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); got != 1 {
		t.Fatalf("draws = %d, want the effect still rendered once", got)
	}
}
