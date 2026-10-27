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

func TestPressureTracksOnlyDuringItsFall(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	ctx.Resources = nil
	start := time.Unix(100, 0)
	actor := ctx.World.Actors[300]
	actor.Moving, actor.MoveStarted, actor.MoveDuration = true, start, 2*time.Second
	actor.FromX, actor.FromY, actor.ToX, actor.ToY = 30, 20, 50, 20
	ctx.World.Actors[300] = actor
	projection := testutil.Projection(800, 600, 20, 20, 0)
	wantX := scene.CellCenter(30 + 10*(31*db.EffectFrameDuration).Seconds())
	for _, ticks := range [][]int{{0, 30, 31, 33}, {0, 30, 33}, {33}} {
		m := &System{}
		m.AddBetween(ctx, Pressure, 300, 2000000, start)
		for _, tick := range ticks {
			now := start.Add(time.Duration(tick) * db.EffectFrameDuration)
			m.Draw(render.NewFrame(800, 600), ctx, projection, now, false)
		}
		effect := m.Active[0]
		if !effect.PressureLanded || math.Abs(effect.ActorOrigin.X-wantX) > 1e-6 {
			t.Fatalf("frames %v: landing X=%f, want %f at the final falling tick", ticks, effect.ActorOrigin.X, wantX)
		}
		// The victim keeps moving, but the landed cross must stay put.
		m.Draw(render.NewFrame(800, 600), ctx, projection, start.Add(time.Second), false)
		if m.Active[0].ActorOrigin != effect.ActorOrigin {
			t.Fatal("landed cross followed the victim")
		}
	}
}

func TestShieldChainCapturesEachSpawnPosition(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	actor := ctx.World.Actors[300]
	actor.Moving, actor.MoveStarted, actor.MoveDuration = true, start, 2*time.Second
	actor.FromX, actor.FromY, actor.ToX, actor.ToY = 30, 20, 50, 20
	ctx.World.Actors[300] = actor
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, ticks := range [][]int{{0, 3, 6, 9, 12}, {12}} {
		m := &System{textures: map[string]*render.Image{
			"__effectfile_effect/shield_boomerang.bmp|blackkey=true": render.NewImage(1, 1),
		}}
		m.AddBetween(ctx, ShieldBoomerang3, 300, 2000000, start)
		for _, tick := range ticks {
			now := start.Add(time.Duration(tick) * db.EffectFrameDuration)
			m.Draw(render.NewFrame(800, 600), ctx, projection, now, false)
		}
		effect := m.Active[0]
		if effect.ShieldChain.Count != 5 {
			t.Fatalf("frames %v: spawned %d shields, want five", ticks, effect.ShieldChain.Count)
		}
		for i, origin := range effect.ShieldChain.Origins {
			wantX := scene.CellCenter(30 + 10*(time.Duration(3*i)*db.EffectFrameDuration).Seconds())
			particle := shieldChainImage(effect, origin, i, 15)
			if math.Abs(particle.Position.X-wantX) > 1e-6 {
				t.Fatalf("frames %v: shield %d crosses X=%f, want spawn position %f", ticks, i, particle.Position.X, wantX)
			}
		}
		origins := effect.ShieldChain.Origins
		m.Draw(render.NewFrame(800, 600), ctx, projection, start.Add(30*db.EffectFrameDuration), false)
		if effect.ShieldChain.Origins != origins {
			t.Fatal("existing shield trajectories moved with the victim")
		}
	}
}

func TestPaladinRealAssetsAndRendering(t *testing.T) {
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		t.Skip("set GORO_DATA_DIR to validate original Paladin assets")
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
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, id := range []int{Guard, ShieldCharge, ShieldBoomer, ReflectShield, HolyCross, GrandCross, Devotion, Providence, CrusaderDef, SpearQuicken, Pressure, Bash3D, BottomGospel, GospelGround, ShieldBoomerang3, Shrink, LineLink} {
		m := &System{}
		spec, ok := Lookup(id)
		if !ok {
			t.Fatalf("missing effect %d", id)
		}
		for _, component := range spec.Components {
			for _, path := range append([]string{component.TextureFile}, component.TextureFiles...) {
				if path != "" && m.EffectFileTexture(manager, path) == nil {
					t.Fatalf("effect %d missing %s", id, path)
				}
			}
			if component.Kind == ComponentFUNC && component.FuncAdapter == EffectFuncUnknown {
				t.Fatalf("effect %d has no renderer for %s", id, component.FuncName)
			}
			if component.StrFile != "" {
				str := m.LoadWorldEffectSTR(manager, component.StrFile, component.TexturePath)
				if str == nil {
					t.Fatalf("missing STR %s", component.StrFile)
				}
				for _, layer := range str.Layers {
					for _, path := range layer.Textures {
						if m.strEffectTexture(manager, path) == nil {
							t.Fatalf("missing STR texture %s", path)
						}
					}
				}
			}
		}
		for _, sound := range spec.Sfx {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Fatalf("effect %d sound: %v", id, err)
			}
		}
		m.AddBetween(ctx, id, 300, 2000000, start)
		drew := false
		for _, tick := range []int{0, 6, 15, 30, 40, 60, 100, 190} {
			frame := render.NewFrame(800, 600)
			now := start.Add(time.Duration(tick) * db.EffectFrameDuration)
			m.Draw(frame, ctx, projection, now, true)
			m.Draw(frame, ctx, projection, now, false)
			v := reflect.ValueOf(frame).Elem()
			drew = drew || v.FieldByName("worldCommands").Len()+v.FieldByName("worldBillboards").Len() > 0
		}
		if len(spec.Components) > 0 && !drew {
			t.Fatalf("effect %d submitted no geometry", id)
		}
	}
}
