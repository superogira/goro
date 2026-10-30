package maps_test

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/internal/testutil"
	maps "github.com/kivutar/goro/game/map"
	worldstate "github.com/kivutar/goro/world"
)

func TestHoveredWalkCellUsesRayWalkCell(t *testing.T) {
	world := worldstate.New()
	world.Player.X = 5
	world.Player.Y = 5
	world.GAT = testutil.FlatWalkableGAT(12, 12)
	projection := testutil.Projection(800, 600, 5.5, 5.5, 0)
	point := projection.Project(6.5, 5.5, 0)

	x, y, ok := maps.HoveredWalkCell(client.Context{World: world}, projection, int(point.X), int(point.Y))
	if !ok || x != 6 || y != 5 {
		t.Fatalf("hovered cell = %d,%d ok=%t, want 6,5 true", x, y, ok)
	}
	if _, _, ok := maps.HoveredWalkCell(client.Context{World: world}, projection, -10000, -10000); ok {
		t.Fatal("hover should not fall back to nearest cell outside the projected map")
	}
}
