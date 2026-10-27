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

func TestRogueParticleRendering(t *testing.T) {
	ctx := testutil.RogueSkillContext()
	projection := testutil.Projection(800, 600, 14, 20, 0)
	starts := time.Unix(100, 0)
	for _, tc := range []struct{ id, tick, count int }{
		{RogueCoin, 56, 120}, {Intimidate, 108, 80}, {TeiHit3, 49, 80},
	} {
		spec, _ := Lookup(tc.id)
		m := &System{textures: map[string]*render.Image{
			"__effectfile_" + spec.Components[0].TextureFile + "|blackkey=true": render.NewImage(8, 8),
		}}
		m.AddAt(ctx, tc.id, 300, starts)
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, starts.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if commands.Len() != 1 || commands.Index(0).FieldByName("Vertices").Len() != tc.count*4 {
			t.Fatalf("effect %d did not batch its native particles", tc.id)
		}
		options := commands.Index(0).FieldByName("Options")
		if !options.FieldByName("DepthTest").Bool() {
			t.Fatalf("effect %d must respect scene depth", tc.id)
		}
		wantBlend := render.BlendLighter
		if tc.id == TeiHit3 {
			wantBlend = render.BlendSourceOver
		}
		if options.FieldByName("Blend").Int() != int64(wantBlend) {
			t.Fatalf("effect %d uses the wrong native blending", tc.id)
		}
	}
}

func TestRogueParticlesNativePhases(t *testing.T) {
	starts := time.Unix(100, 0)
	for _, tc := range []struct {
		id, count, before, peak, end int
	}{
		{RogueCoin, 120, 17, 56, 108},
		{Intimidate, 80, 69, 108, 146},
		{TeiHit3, 80, 29, 49, 79},
	} {
		effect := Instance{EffectID: tc.id, Starts: starts, ActorID: 100}
		visible := 0
		for i := 0; i < tc.count; i++ {
			for _, frame := range []int{tc.before, tc.end} {
				if _, tint := rogueBurstParticle(effect, i, frame); tint.A != 0 {
					t.Fatalf("effect %d particle %d visible outside its native phase at tick %d", tc.id, i, frame)
				}
			}
			corners, tint := rogueBurstParticle(effect, i, tc.peak)
			if tint.A > 0 {
				visible++
				for _, p := range corners {
					if !scene.IsFinite(p.X) || !scene.IsFinite(p.Y) || !scene.IsFinite(p.Z) {
						t.Fatal("non-finite particle geometry")
					}
				}
			}
		}
		if visible != tc.count {
			t.Fatalf("effect %d: %d visible particles, want %d", tc.id, visible, tc.count)
		}
	}
	effect := Instance{EffectID: BackStab, Starts: starts, ProjectileDirection: scene.Point3{X: 6}}
	for i := 0; i < 80; i++ {
		if _, _, _, alpha := impactParticle(effect, i, 30); alpha != 0 {
			t.Fatal("Back Stab appeared before its delayed particle launch")
		}
		position, size, _, alpha := impactParticle(effect, i, 49)
		if alpha <= 0 || position.X <= 0 || math.Abs(position.Z) > position.X || size <= 0 {
			t.Fatalf("Back Stab must travel in the attack direction: %+v, %g, %d", position, size, alpha)
		}
		if _, _, _, alpha := impactParticle(effect, i, 125); alpha != 0 {
			t.Fatal("Back Stab did not fade out")
		}
	}
}

func TestBackStab2008Opacity(t *testing.T) {
	effect := Instance{EffectID: BackStab, Starts: time.Unix(100, 0)}
	for i := 0; i < 80; i++ {
		for _, tc := range []struct{ age, alpha int }{{1, 0}, {2, 15}, {10, 135}, {11, 132}, {54, 3}, {55, 0}} {
			frame := tc.age + 39 - HitEffectRandom(effect, i*5, 10)
			_, _, _, alpha := impactParticle(effect, i, frame)
			if alpha != tc.alpha {
				t.Fatalf("particle %d at age %d: alpha = %d, want 2008 alpha %d", i, tc.age, alpha, tc.alpha)
			}
		}
	}
}

func TestRogueRealAssets(t *testing.T) {
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
	for _, id := range []int{StealCoin, RogueCoin, Intimidate, BackStab, TeiHit3, StripWeapon, StripShield, StripArmor, StripHelm, QuakeBody4, NPCStop2} {
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
	if m.EffectFileTexture(manager, "effect/alpabet.bmp") == nil {
		t.Fatal("missing Scribble bitmap alphabet")
	}
}
