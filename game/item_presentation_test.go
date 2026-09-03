package game

import (
	"math"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func TestFallingGroundItemKeepsShadowOnTerrain(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Items[1] = worldstate.FloorItem{
		ID:        1,
		ItemID:    501,
		X:         10,
		Y:         20,
		Falling:   true,
		DroppedAt: now,
	}
	projection := newSceneProjectionForTarget(800, 600, cellCenter(10), cellCenter(20), 0)
	mode := &WorldMode{}
	entries := mode.collectSceneItemEntries(render.NewFrame(800, 600), client.Context{World: world}, projection, now)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	entry := entries[0]
	if entry.worldZ <= entry.groundZ {
		t.Fatalf("falling item z = %.2f, ground z = %.2f", entry.worldZ, entry.groundZ)
	}
	wantDepth := projection.Depth(entry.worldX, entry.worldY, entry.groundZ+actorShadowTerrainLift)
	if math.Abs(entry.shadowDepth-wantDepth) > 1e-9 {
		t.Fatalf("shadow depth = %.9f, want terrain depth %.9f", entry.shadowDepth, wantDepth)
	}
}

func TestGroundItemSortsAfterItsShadow(t *testing.T) {
	world := worldstate.New()
	world.Items[1] = worldstate.FloorItem{ID: 1, ItemID: 501, X: 10, Y: 20}
	projection := newSceneProjectionForTarget(800, 600, cellCenter(10), cellCenter(20), 0)
	entries := (&WorldMode{}).collectSceneItemEntries(
		render.NewFrame(800, 600),
		client.Context{World: world},
		projection,
		time.Now(),
	)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if entries[0].depth >= entries[0].shadowDepth {
		t.Fatalf("item depth = %.4f, shadow depth = %.4f; item must sort after its shadow", entries[0].depth, entries[0].shadowDepth)
	}
}

func TestDrawGroundItemShadowUsesSharedOriginalShadowSprite(t *testing.T) {
	if groundItemShadowScale != 0.25 {
		t.Fatalf("floor-item shadow scale = %.2f, want roBrowser's 0.25", groundItemShadowScale)
	}
	shadowView := &spriteView{
		spr: &res.SPR{
			RGBAIndex: 0,
			Frames: []res.SPRFrame{{
				Type:   res.SPRFrameRGBA,
				Width:  16,
				Height: 8,
				Data:   solidRGBAFrame(16, 8),
			}},
		},
		act: &res.ACT{Actions: []res.ACTAction{{
			Animations: []res.ACTAnimation{{Layers: []res.ACTLayer{{
				Index:   0,
				SPRType: res.SPRFrameRGBA,
				ScaleX:  1,
				ScaleY:  1,
				Color:   [4]float32{1, 1, 1, 1},
			}}}},
		}}},
		images:     make(map[spriteFrameKey]*render.Image),
		billboards: make(map[singleSpriteBillboardKey]*spriteBillboard),
	}
	mode := &WorldMode{shadowView: shadowView}
	entry := sceneItemDrawEntry{
		worldX:      cellCenter(10),
		worldY:      cellCenter(20),
		groundZ:     0,
		shadowScale: groundItemShadowScale,
		shadow:      1,
	}
	projection := newSceneProjectionForTarget(800, 600, entry.worldX, entry.worldY, 0)
	if !mode.drawGroundItemShadowEntry3D(render.NewFrame(800, 600), projection, entry) {
		t.Fatal("floor-item shadow was not drawn")
	}
}

func TestGroundItemShadowIsOffsetDownWithoutMutatingSharedBillboard(t *testing.T) {
	base := &spriteBillboard{anchorX: 20, anchorY: 30}
	got := groundItemShadowBillboard(base)
	if got == base {
		t.Fatal("item shadow offset should copy the shared billboard")
	}
	if got.anchorX != 20 || got.anchorY != 30-groundItemShadowOffsetY {
		t.Fatalf("item shadow anchor = %.1f, %.1f", got.anchorX, got.anchorY)
	}
	if base.anchorX != 20 || base.anchorY != 30 {
		t.Fatalf("shared shadow billboard was mutated: %.1f, %.1f", base.anchorX, base.anchorY)
	}
}
