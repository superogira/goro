package effects

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

func TestWizardStormGustSimplifiedEffects(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	m := &System{textures: map[string]*render.Image{
		"__effectfile_effect/ice.tga|blackkey=true": render.NewImage(32, 32),
	}}
	start := time.Unix(100, 0)
	spec, _ := Lookup(StormGust)
	for _, simplified := range []bool{false, true, false} {
		ctx.Session.LessEffects = simplified
		frame := render.NewFrame(800, 600)
		m.drawGroundSpikes(frame, ctx, spec.Components[1], Instance{EffectID: StormGust, Starts: start},
			scene.Point3{}, start.Add(60*db.EffectFrameDuration))
		got := reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len()
		if (got == 0) != simplified {
			t.Fatalf("simplified=%t: got %d spike batches", simplified, got)
		}
		wantSTR := "stormgust"
		if simplified {
			wantSTR = "storm_min"
		}
		if got := ResolveEffectSTRFile(spec.Components[0], Instance{}, lessEffectsEnabled(ctx)); got != wantSTR {
			t.Fatalf("simplified=%t: STR=%q, want %q", simplified, got, wantSTR)
		}
	}
}

func TestWizardNativeGeometry(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	texture := render.NewImage(32, 32)
	m := &System{textures: map[string]*render.Image{
		"__effectfile_effect/ice.tga|blackkey=true":   texture,
		"__effectfile_effect/stone.bmp|blackkey=true": texture,
	}}
	start := time.Unix(100, 0)
	for _, tc := range []struct{ id, tick, spikes int }{
		{IceWall, 30, 3}, {IceWall, 10000, 3},
		{EarthSpike, 20, 7}, {HeavenDrive, 20, 25},
		{StormGust, 57, 0}, {StormGust, 58, 2}, {StormGust, 145, 8}, {StormGust, 215, 0},
	} {
		frame := render.NewFrame(800, 600)
		spec, _ := Lookup(tc.id)
		component := spec.Components[len(spec.Components)-1]
		m.drawGroundSpikes(frame, ctx, component, Instance{EffectID: tc.id, Starts: start}, scene.Point3{}, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		cmds := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if tc.spikes == 0 {
			if cmds.Len() != 0 {
				t.Fatalf("effect %d tick %d emitted spikes too early or late", tc.id, tc.tick)
			}
			continue
		}
		if cmds.Len() != 1 || cmds.Index(0).FieldByName("Vertices").Len() != tc.spikes*12 {
			t.Fatalf("effect %d tick %d: expected %d four-sided spikes", tc.id, tc.tick, tc.spikes)
		}
		options := cmds.Index(0).FieldByName("Options")
		if options.FieldByName("AfterBillboards").Bool() != (tc.id == IceWall) ||
			!options.FieldByName("DepthTest").Bool() || options.FieldByName("DepthWrite").Bool() {
			t.Fatalf("effect %d: incorrect translucent spike depth/order", tc.id)
		}
		vertices := cmds.Index(0).FieldByName("Vertices")
		for i := 0; i < vertices.Len(); i += 3 {
			if vertices.Index(i+1).FieldByName("Y").Float() <= vertices.Index(i).FieldByName("Y").Float() {
				t.Fatal("native spike points down")
			}
		}
	}
}

func TestWizardFirePillarStaysUpright(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	m := &System{textures: map[string]*render.Image{
		"__effectfile_effect/magic_red.tga|blackkey=true": render.NewImage(32, 32),
	}}
	start := time.Unix(100, 0)
	m.AddCellLifetime(ctx, FirePillarOn, 900, 15, 20, start, 0, true)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, tick := range []int{0, 60, 300} {
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
		cmds := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if cmds.Len() != 3 {
			t.Fatal("Fire Pillar must have three spinning bands")
		}
		for i := 0; i < cmds.Len(); i++ {
			vertices := cmds.Index(i).FieldByName("Vertices")
			bottom, top := vertices.Index(0).FieldByName("Y").Float(), vertices.Index(1).FieldByName("Y").Float()
			if top <= bottom {
				t.Fatal("Fire Pillar points underground")
			}
			for j := 2; j < vertices.Len(); j += 2 {
				if math.Abs(vertices.Index(j).FieldByName("Y").Float()-bottom) > 1e-5 ||
					math.Abs(vertices.Index(j+1).FieldByName("Y").Float()-top) > 1e-5 {
					t.Fatal("Fire Pillar must rotate without tilting")
				}
			}
		}
	}
}

func TestWizardJupitelTravelAndImpact(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	m := &System{textures: make(map[string]*render.Image)}
	for _, id := range []int{JupitelThunder, JupitelHit} {
		spec, _ := Lookup(id)
		c := spec.Components[0]
		for _, path := range append([]string{c.TextureFile}, c.TextureFiles...) {
			m.textures["__effectfile_"+path+"|blackkey=true"] = render.NewImage(32, 32)
		}
	}
	m.AddBetween(ctx, JupitelThunder, 300, 2000000, start)
	effect := m.Active[0]
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, tick := range []int{0, 10, 19} {
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
		cmds := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if cmds.Len() != 2 {
			t.Fatal("Jupitel needs a core and animated shell")
		}
		expected := effect.ActorOrigin.X + effect.ProjectileDirection.X*float64(tick+1)/20
		if math.Abs(cmds.Index(0).FieldByName("Center").Index(0).Float()-expected) > 1e-5 || !cmds.Index(0).FieldByName("Options").FieldByName("DepthTest").Bool() {
			t.Fatal("Jupitel must travel from the caster with depth testing")
		}
	}
	m.Active = nil
	m.AddAt(ctx, JupitelHit, 300, start)
	for _, tick := range []int{0, 10, 20, 240} {
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
		cmds := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		want := 1
		if tick >= 20 {
			want = 2
		}
		if cmds.Len() != want {
			t.Fatalf("Jupitel impact tick %d: lost its flashes or plasma", tick)
		}
	}
}
