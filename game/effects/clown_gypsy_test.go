package effects

import (
	"image/color"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

func TestClownGypsyBodyEffectRendering(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	start := time.Unix(100, 0)
	for _, tc := range []struct{ id, frame, copies int }{{PinkBody, 1000, 2}, {ChemicalBody, 30, 2}, {ChemicalBody, 31, 1}} {
		m := &System{}
		m.AddAt(ctx, tc.id, 300, start)
		frame := render.NewFrame(800, 600)
		billboard := &scene.Billboard{Image: render.NewImage(10, 20), AnchorX: 5, AnchorY: 20}
		entry := scene.ActorDrawEntry{Actor: ctx.World.Actors[300], WorldX: 20.5, WorldY: 20.5}
		m.DrawActor(ctx, frame, testutil.Projection(800, 600, 15, 20, 0), entry, billboard, 1, 1, 1,
			color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tc.frame)*db.EffectFrameDuration), 0)
		if count := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); count != tc.copies {
			t.Fatalf("body effect %d frame %d: %d layers, want %d", tc.id, tc.frame, count, tc.copies)
		}
	}
}
