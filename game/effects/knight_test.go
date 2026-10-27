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

func TestKnightBowlingRings(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	starts := time.Unix(100, 0)
	spec, _ := Lookup(BowlingBash)
	component := spec.Components[0]
	m := &System{textures: map[string]*render.Image{}}
	for _, path := range []string{component.TextureFile, component.TextureFiles[0]} {
		m.textures["__effectfile_"+path+"|blackkey=true"] = render.NewImage(8, 8)
	}
	m.AddAt(ctx, BowlingBash, 300, starts)
	for _, tc := range []struct{ tick, draws int }{{0, 2}, {5, 3}, {20, 1}, {50, 0}} {
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, scene.Projection{}, starts.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if commands.Len() != tc.draws {
			t.Fatalf("tick %d: %d draws, want %d", tc.tick, commands.Len(), tc.draws)
		}
		for i := 0; i < commands.Len(); i++ {
			command := commands.Index(i)
			options := command.FieldByName("Options")
			if options.FieldByName("DepthWrite").Bool() || options.FieldByName("Blend").Int() != int64(render.BlendSourceOver) {
				t.Fatal("Bowling Bash must alpha blend without writing depth")
			}
			if options.FieldByName("DepthTest").Bool() != (i < commands.Len()-1) {
				t.Fatal("only the yellow ground ring bypasses depth testing")
			}
			vertices := command.FieldByName("Vertices")
			if vertices.Len() != 22 || command.FieldByName("Indices").Len() != 60 {
				t.Fatal("Bowling Bash rings must have ten faces")
			}
			var center scene.Point3
			for j := 0; j < 10; j++ {
				vertex := vertices.Index(j * 2)
				center.X += vertex.FieldByName("X").Float() / 10
				center.Z += vertex.FieldByName("Z").Float() / 10
			}
			if math.Abs(center.X-30.5) > .001 || math.Abs(center.Z-20.5) > .001 {
				t.Fatal("Bowling Bash followed the victim after impact")
			}
		}
		actor := ctx.World.Actors[300]
		actor.X += 10
		actor.Dir = (actor.Dir + 2) % 8
		ctx.World.Actors[300] = actor
	}
}

func TestKnightRealAssetsAndRendering(t *testing.T) {
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		t.Skip("set GORO_DATA_DIR to check real Knight assets")
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
	m := &System{}
	for _, id := range []int{Pierce, PierceSelf, BrandishSpear, BrandishSpear2, BowlingBash, BowlingSelf, SpearStabSelf, SpearBmrSelf, TwoHandQuicken, AutoCounter, SpearBoomerang} {
		spec, _ := Lookup(id)
		for _, component := range spec.Components {
			for _, path := range append([]string{component.TextureFile}, component.TextureFiles...) {
				if path != "" && m.EffectFileTexture(manager, path) == nil {
					t.Fatalf("effect %d missing texture %s", id, path)
				}
			}
			if component.TextureName != "" && m.effectTexture(manager, component.TextureName) == nil {
				t.Fatalf("effect %d missing texture %s", id, component.TextureName)
			}
			if component.StrFile != "" {
				str := m.LoadWorldEffectSTR(manager, component.StrFile, component.TexturePath)
				if str == nil {
					t.Fatalf("missing STR %s", component.StrFile)
				}
				for _, layer := range str.Layers {
					for _, path := range layer.Textures {
						if m.strEffectTexture(manager, path) == nil {
							t.Fatalf("STR %s missing %s", component.StrFile, path)
						}
					}
				}
			}
		}
		for _, sound := range spec.Sfx {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Fatal(err)
			}
		}
	}
	starts := time.Unix(100, 0)
	m.AddBetween(ctx, SpearBoomerang, 300, 2000000, starts)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for i, tick := range []int{0, 4, 8, 12} {
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, starts.Add(time.Duration(tick)*db.EffectFrameDuration), false)
		if count := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); count != i+1 {
			t.Fatalf("tick %d drew %d spears, want %d", tick, count, i+1)
		}
	}
}
