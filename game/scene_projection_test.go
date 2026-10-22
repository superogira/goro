package game

import (
	"math"
	"testing"

	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func TestBilinearHeight(t *testing.T) {
	heights := [4]float32{0, 10, 20, 30}
	if got := scene.BilinearHeight(heights, 0.5, 0.5); got != 15 {
		t.Fatalf("height = %v, want 15", got)
	}
}

func TestTerrainHeightPrefersGAT(t *testing.T) {
	world := &worldstate.World{
		GAT: &res.GAT{
			Width:  1,
			Height: 1,
			Cells:  []res.GATCell{{Heights: [4]float32{10, 20, 30, 40}}},
		},
		GND: &res.GND{
			Width:  1,
			Height: 1,
			Cells:  []res.GNDCell{{Heights: [4]float32{1, 1, 1, 1}}},
		},
	}
	if got := scene.TerrainHeightAt(world, 0, 0); got != 25 {
		t.Fatalf("terrain height = %v, want GAT center height 25", got)
	}
}

func TestCameraProjectionCentersPlayerCell(t *testing.T) {
	projection := testutil.Projection(800, 600, 10.5, 20.5, 5)
	point := projection.Project(10.5, 20.5, 5)
	if math.Abs(float64(point.X)-400) > 0.01 || math.Abs(float64(point.Y)-300) > 0.01 {
		t.Fatalf("projected point = %.1f, %.1f, want 400, 300", point.X, point.Y)
	}
}

func TestSceneCameraDefaultZoomUsesGameplayScale(t *testing.T) {
	if got := scene.CameraZoom() * 0.5; got != 62.5 {
		t.Fatalf("default camera distance = %.1f, want reference client default 62.5", got)
	}
}

func TestCameraProjectionUsesRequestedPitch(t *testing.T) {
	low := scene.NewProjection(800, 600, 10.5, 20.5, 5, 0, scene.DefaultCameraMinPitch, 125)
	high := scene.NewProjection(800, 600, 10.5, 20.5, 5, 0, scene.DefaultCameraMaxPitch, 125)
	lowEye, _, _, _ := low.CameraBasis()
	highEye, _, _, _ := high.CameraBasis()

	if low.CameraPitch != scene.DefaultCameraMinPitch || high.CameraPitch != scene.DefaultCameraMaxPitch {
		t.Fatalf("projection pitches = %.1f and %.1f, want %.1f and %.1f", low.CameraPitch, high.CameraPitch, scene.DefaultCameraMinPitch, scene.DefaultCameraMaxPitch)
	}
	if highEye.Y <= lowEye.Y {
		t.Fatalf("high-pitch eye y = %.2f, want above low-pitch eye y %.2f", highEye.Y, lowEye.Y)
	}
}

func TestCameraProjectionMapsPositiveXToScreenRight(t *testing.T) {
	projection := testutil.Projection(800, 600, 10.5, 20.5, 5)
	center := projection.Project(10.5, 20.5, 5)
	right := projection.Project(11.5, 20.5, 5)
	if right.X <= center.X {
		t.Fatalf("positive world X projected to %.1f, center %.1f; want screen right", right.X, center.X)
	}
}

func TestCameraProjectionMovesHigherWorldPointUpScreen(t *testing.T) {
	projection := testutil.Projection(800, 600, 10.5, 20.5, 5)
	base := projection.Project(10.5, 20.5, 5)
	higher := projection.Project(10.5, 20.5, 15)
	if higher.Y >= base.Y {
		t.Fatalf("higher point y = %.1f, base y = %.1f, want higher point above base", higher.Y, base.Y)
	}
}

func TestBillboardBasisStaysScreenAlignedOffCenter(t *testing.T) {
	projection := testutil.ProjectionYawZoom(800, 600, 10.5, 20.5, 5, 0, 150)
	worldX, worldY, worldZ := 18.5, 20.5, 5.0
	right, up, unitsPerPixel, ok := projection.BillboardBasis(worldX, worldY, worldZ)
	if !ok {
		t.Fatal("missing billboard basis")
	}

	center := scene.Point3{X: worldX, Y: worldZ, Z: worldY}
	project := func(p scene.Point3) scene.Point2 {
		return projection.Project(p.X, p.Z, p.Y)
	}
	left := project(scene.Add3(center, scene.Mul3(right, -32*unitsPerPixel)))
	rightPoint := project(scene.Add3(center, scene.Mul3(right, 32*unitsPerPixel)))
	top := project(scene.Add3(center, scene.Mul3(up, 64*unitsPerPixel)))
	bottom := project(scene.Add3(center, scene.Mul3(up, -64*unitsPerPixel)))
	centerPoint := project(center)

	if math.Abs(float64(left.Y-rightPoint.Y)) > 0.25 {
		t.Fatalf("billboard horizontal edge y mismatch: left %.3f right %.3f", left.Y, rightPoint.Y)
	}
	if math.Abs(float64(top.X-centerPoint.X)) > 0.25 || math.Abs(float64(bottom.X-centerPoint.X)) > 0.25 {
		t.Fatalf("billboard vertical edge x mismatch: top %.3f center %.3f bottom %.3f", top.X, centerPoint.X, bottom.X)
	}
	if got := math.Abs(float64(rightPoint.X - left.X)); math.Abs(got-64) > 0.5 {
		t.Fatalf("billboard projected width = %.3f, want 64", got)
	}
	if got := math.Abs(float64(bottom.Y - top.Y)); math.Abs(got-128) > 0.5 {
		t.Fatalf("billboard projected height = %.3f, want 128", got)
	}
}

func TestCameraDepthOrdersFarBeforeNear(t *testing.T) {
	projection := testutil.Projection(800, 600, 10.5, 20.5, 5)
	near := projection.Depth(10.5, 20.5, 5)
	far := projection.Depth(10.5, 30.5, 5)
	if far <= near {
		t.Fatalf("far depth %.2f must be greater than near depth %.2f for painter sorting", far, near)
	}
}
