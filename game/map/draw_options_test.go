package maps

import (
	"math"
	"testing"

	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

func TestTriangleDrawOptionsEnableDepthTest(t *testing.T) {
	options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	if options.Filter != render.FilterLinear || options.Address != render.AddressRepeat {
		t.Fatalf("options filter/address = %v/%v", options.Filter, options.Address)
	}
	if !options.DepthTest {
		t.Fatal("triangle options should enable depth testing")
	}
}

func TestGroundTextureDrawOptionsClampTileEdges(t *testing.T) {
	options := groundTextureDrawOptions()
	if options.Filter != render.FilterLinear {
		t.Fatalf("ground filter = %v, want linear", options.Filter)
	}
	if options.Address != render.AddressClampToZero {
		t.Fatalf("ground address = %v, want clamp", options.Address)
	}
	if !options.DepthTest || !options.DepthWrite {
		t.Fatalf("ground depth flags test=%t write=%t, want both true", options.DepthTest, options.DepthWrite)
	}
}

func TestRSMModelDrawOptionsUseTinyDepthBias(t *testing.T) {
	options := rsmModelDrawOptions(render.FilterLinear, render.AddressClampToEdge)
	if options.Filter != render.FilterLinear || options.Address != render.AddressClampToEdge {
		t.Fatalf("RSM options filter/address = %v/%v", options.Filter, options.Address)
	}
	if !options.DepthTest || !options.DepthWrite {
		t.Fatalf("RSM depth flags test=%t write=%t, want both true", options.DepthTest, options.DepthWrite)
	}
	if math.Abs(float64(options.DepthBias-rsmModelDepthBias)) > 0.0000001 || options.DepthBias <= 0 {
		t.Fatalf("RSM depth bias = %.10f, want %.10f", options.DepthBias, rsmModelDepthBias)
	}
}
