package effects

import (
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/render"
)

func TestThiefEffectsWithClientAssets(t *testing.T) {
	manager := testutil.RealDataManager(t)
	ctx := testutil.ThiefEffectContext()
	ctx.Resources = manager
	m := &System{}
	starts := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 14, 20, 0)
	for _, tc := range []struct{ id, frame, count int }{
		{Steal, 20, 10}, {Detoxication, 20, 5}, {Detoxication, 98, 1},
		{SprinkleSand, 20, 80}, {ThrowItem3, 6, 4},
	} {
		m.Active = nil
		m.AddBetween(ctx, tc.id, 300, 2000000, starts)
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, starts.Add(time.Duration(tc.frame)*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if commands.Len() != tc.count {
			t.Fatalf("effect %d tick %d: want %d particles, got %d", tc.id, tc.frame, tc.count, commands.Len())
		}
		spec, _ := Lookup(tc.id)
		for _, sound := range spec.Sfx {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Error(err)
			}
		}
	}
}

func TestThiefParticleMotion(t *testing.T) {
	effect := Instance{EffectID: Steal, Starts: time.Unix(100, 0)}
	for i := 0; i < 10; i++ {
		first, middle, last := stealParticleAt(effect, i, 0), stealParticleAt(effect, i, 24), stealParticleAt(effect, i, 48)
		if middle.Position.Y <= first.Position.Y || last.Position.Y >= middle.Position.Y || last.size >= first.size || last.Alpha >= first.Alpha {
			t.Fatalf("Steal particle %d must rise, fall, shrink and fade: %+v / %+v / %+v", i, first, middle, last)
		}
		if p := stealParticleAt(effect, i, 49); p.Alpha != 0 || p.size != 0 {
			t.Fatal("Steal must finish at fifty ticks")
		}
	}
	effect.EffectID = Detoxication
	first, last := RisingParticleAt(effect, 0, 0), RisingParticleAt(effect, 0, 38)
	if first.Alpha <= 0 || last.Position.Y <= first.Position.Y || last.Angle <= first.Angle || RisingParticleAt(effect, 0, 39).Alpha != 0 {
		t.Fatal("Detoxify particles must rise, rotate and fade over forty ticks")
	}
	effect.EffectID = SprinkleSand
	for i := 0; i < 80; i++ {
		if sprinkleSandParticleAt(effect, i, 0).Alpha != 0 || sprinkleSandParticleAt(effect, i, 20).Alpha <= 0 || sprinkleSandParticleAt(effect, i, 65).Alpha != 0 {
			t.Fatalf("Sand Attack particle %d has wrong delay or fade", i)
		}
	}
}
