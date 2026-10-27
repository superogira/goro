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

func TestSonicBlowRingKeepsItsScreenDimensions(t *testing.T) {
	ctx := testutil.AssassinEffectContext()
	m := &System{textures: map[string]*render.Image{"__effectfile_effect/alpha_center.tga|blackkey=true": render.NewImage(8, 8)}}
	starts := time.Unix(100, 0)
	spec, _ := Lookup(SonicBlow)
	for _, center := range []scene.Point3{{X: 10.5, Z: 20.5}, {X: 10.5, Z: 35.5}} {
		projection := testutil.Projection(800, 600, 14, 20, 0)
		frame := render.NewFrame(800, 600)
		m.drawSonicBlowEffect(frame, ctx, projection, spec.Components[0], Instance{Starts: starts}, center.X, center.Z, center.Y, starts.Add(9*db.EffectFrameDuration))
		command := reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Index(0)
		if command.FieldByName("Options").FieldByName("DepthTest").Bool() {
			t.Fatal("Sonic Blow screen ring must not be hidden by scene geometry")
		}
		vertices := command.FieldByName("Vertices")
		origin := projection.Project(center.X, center.Z, center.Y)
		for i, radius := range []float64{50.375, 60.375} { // ten native radius updates
			v := vertices.Index(i)
			point := projection.Project(v.FieldByName("X").Float(), v.FieldByName("Z").Float(), v.FieldByName("Y").Float())
			if math.Abs(float64(point.X-origin.X)-radius) > .001 || math.Abs(float64(point.Y-origin.Y)+20) > .001 {
				t.Fatalf("ring point %+v, caster %+v: want radius %g and height 20 pixels", point, origin, radius)
			}
		}
	}
}

func TestAssassinPoisonRotationMatchesSakexe(t *testing.T) {
	ctx := testutil.AssassinEffectContext()
	starts := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 14, 20, 0)
	for _, id := range []int{EnchantPoison, VenomDust2} {
		spec, _ := Lookup(id)
		m := &System{EffectViews: map[string]*scene.SpriteView{spec.Components[0].SpriteFile: testutil.EffectRenderTestSprite()}}
		m.AddAt(ctx, id, 300, starts)
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, starts, false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if commands.Len() != 1 {
			t.Fatalf("effect %d: expected one initial particle, got %d", id, commands.Len())
		}
		command := commands.Index(0)
		center, axis := command.FieldByName("Center"), command.FieldByName("RightAxis")
		_, up, _, _ := projection.BillboardBasis(center.Index(0).Float(), center.Index(2).Float(), center.Index(1).Float())
		right := scene.Point3{X: axis.Index(0).Float(), Y: axis.Index(1).Float(), Z: axis.Index(2).Float()}
		// Sakexe's positive roll turns the sprite's right axis down the screen.
		if vertical := scene.Dot3(right, up); vertical >= 0 {
			t.Errorf("effect %d: particle rotates upward instead of downward: %g", id, vertical)
		}
	}
}

func TestAssassinRealEffectAssets(t *testing.T) {
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
	for _, id := range []int{Cloaking, SonicBlow, SonicBlow2, SonicBlowHit, Grimtooth, GrimtoothAtk, EnchantPoison, PoisonReact, VenomDust, VenomDust2, VenomSplasher, ThrowItem6} {
		spec, _ := Lookup(id)
		for _, c := range spec.Components {
			if c.TextureFile != "" && m.EffectFileTexture(manager, c.TextureFile) == nil {
				t.Errorf("effect %d missing texture %s", id, c.TextureFile)
			}
			if c.SpriteFile != "" && m.EffectSpriteView(manager, c.SpriteFile) == nil {
				t.Errorf("effect %d missing sprite %s", id, c.SpriteFile)
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
	ctx := testutil.AssassinEffectContext()
	ctx.Resources = manager
	starts := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 14, 20, 0)
	for _, id := range []int{SonicBlow, Grimtooth, GrimtoothAtk, EnchantPoison, VenomDust2, ThrowItem6} {
		m.Active = nil
		m.AddBetween(ctx, id, 300, 2000000, starts)
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, starts.Add(6*db.EffectFrameDuration), false)
		value := reflect.ValueOf(frame).Elem()
		if value.FieldByName("worldCommands").Len()+value.FieldByName("worldBillboards").Len() == 0 {
			t.Errorf("effect %d loaded its assets but rendered nothing", id)
		}
	}
}
