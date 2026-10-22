package effects

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/render"
)

func TestSageGroundGeometry(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	texture := render.NewImage(64, 64)
	m := &System{textures: map[string]*render.Image{
		"__effectfile_effect/ring_red.tga|blackkey=true":    texture,
		"__effectfile_effect/magic_green.tga|blackkey=true": texture,
		"__effectfile_effect/aaa copy.bmp|blackkey=true":    texture,
	}}
	projection := testutil.Projection(800, 600, 0, 0, 0)
	for _, tc := range []struct {
		id, tick, vertices int
		alpha, maxU        float64
		additive           bool
	}{
		{Volcano, 9, 320, 200.0 / 255, 20.0 / 21, false},
		{Volcano, 109, 0, 0, 0, false},
		{BottomViolent, 58, 40, 0, 10.0 / 21, false},
		{BottomLand, 0, 4, 100.0 / 255, 1, true},
		{BottomLand, 120, 4, 100.0 / 255, 1, true},
	} {
		frame := render.NewFrame(800, 600)
		spec, _ := Lookup(tc.id)
		effect := Instance{EffectID: tc.id, Starts: start}
		m.DrawWorldEffectComponent(frame, ctx, projection, effect, spec.Components[0], 0, 0, 0, 0,
			0, spec.Duration, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		cmds := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if tc.vertices == 0 {
			if cmds.Len() != 0 {
				t.Fatal("ground activation remains visible after fading")
			}
			continue
		}
		if cmds.Len() != 1 || cmds.Index(0).FieldByName("Vertices").Len() != tc.vertices {
			t.Fatalf("effect %d: wrong ring/tile geometry", tc.id)
		}
		cmd := cmds.Index(0)
		options := cmd.FieldByName("Options")
		if !options.FieldByName("DepthTest").Bool() ||
			(options.FieldByName("Blend").Int() == int64(render.BlendLighter)) != tc.additive {
			t.Fatalf("effect %d: wrong blending/depth", tc.id)
		}
		vertices := cmd.FieldByName("Vertices")
		maxU := 0.0
		for i := 0; i < vertices.Len(); i++ {
			vertex := vertices.Index(i)
			maxU = math.Max(maxU, vertex.FieldByName("SrcX").Float()/64)
			if tc.alpha != 0 && math.Abs(vertex.FieldByName("ColorA").Float()-tc.alpha) > 1e-6 {
				t.Fatalf("effect %d: wrong opacity", tc.id)
			}
			if tc.id == BottomLand {
				x, y, z := vertex.FieldByName("X").Float(), vertex.FieldByName("Y").Float(), vertex.FieldByName("Z").Float()
				radius := math.Hypot(x, z)
				if math.Abs(y-.4) > 1e-6 || radius < 1.12 || radius > 1.4 || math.Abs(x*z) > 1e-6 {
					t.Fatal("Land Protector must pulse as a raised diamond with native dimensions")
				}
				if i == 0 && (x != 0 || z >= 0 || vertex.FieldByName("SrcY").Float() != 0) {
					t.Fatal("Land Protector texture is flipped")
				}
			}
		}
		if math.Abs(maxU-tc.maxU) > 1e-6 {
			t.Fatalf("effect %d: texture stretched across the wrong portion of the ring", tc.id)
		}
	}
}

func TestSageGroundRingTiming(t *testing.T) {
	// Compare time sampling to the native float32 PrimGI4 state machine,
	// including its initial random fade and many consecutive resets.
	for initial := 0; initial < 100; initial++ {
		radius, angle, alpha := float32(2.6), 80, initial
		for n := 1; n <= 2000; n++ {
			radius += .04
			angle--
			if angle < 10 {
				angle, alpha = 10, 0
			}
			if radius >= 2.5 {
				alpha -= 2
				if alpha <= 0 {
					alpha, radius, angle = 0, float32(2.5)-float32(.36), 74
				}
			} else {
				alpha += 20
			}
			gotRadius, gotAngle, gotAlpha := propertyGroundRing(n, initial)
			if math.Abs(gotRadius-float64(radius)) > 1e-5 || gotAngle != float64(angle) || gotAlpha != alpha {
				t.Fatalf("initial %d tick %d: got (%f,%f,%d), want (%f,%d,%d)",
					initial, n, gotRadius, gotAngle, gotAlpha, radius, angle, alpha)
			}
		}
	}
}
