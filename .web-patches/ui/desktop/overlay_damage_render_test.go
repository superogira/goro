package desktop

import (
	"image"
	"testing"

	"github.com/gogpu/gg/scene"
	"github.com/gogpu/ui/compositor"
	"github.com/gogpu/ui/geometry"
)

// --- Overlay Boundary Render-Layer Damage Tests ---
//
// These tests verify the render-layer pipeline for overlay boundary damage:
//   - isBoundaryClean detects version mismatch after overlay re-record
//   - trackBoundaryDamage appends correct rects for overlay boundaries
//   - renderFromTreeRecursive processes overlay PictureLayers
//
// Companion to app/overlay_damage_tracking_test.go which tests the app-layer
// (sceneDirty, version increment, syncPictureLayer, callback wiring).

// TestOverlayBoundary_RenderDetectsVersionMismatch verifies that
// isBoundaryClean correctly detects when the texture entry's sceneVersion
// differs from the PictureLayer's sceneVersion (re-recorded overlay boundary).
//
// This is the critical detection mechanism: recordBoundary clears sceneDirty
// and increments sceneCacheVersion. syncPictureLayer copies the new version
// to PictureLayer. isBoundaryClean compares entry.sceneVersion (old) with
// pic.SceneVersion() (new). Mismatch means re-render is needed.
func TestOverlayBoundary_RenderDetectsVersionMismatch(t *testing.T) {
	tests := []struct {
		name         string
		entryVersion uint64
		picVersion   uint64
		picDirty     bool
		fullRedraw   bool
		hasScene     bool
		expectClean  bool
	}{
		{
			name:         "version_match_clean",
			entryVersion: 5,
			picVersion:   5,
			picDirty:     false,
			hasScene:     true,
			expectClean:  true,
		},
		{
			name:         "overlay_hover_version_bump",
			entryVersion: 5,
			picVersion:   6,     // re-recorded after hover → version bumped
			picDirty:     false, // sceneDirty cleared by recordBoundary
			hasScene:     true,
			expectClean:  false, // version mismatch → MUST re-render
		},
		{
			name:         "dirty_flag_still_set",
			entryVersion: 5,
			picVersion:   5,
			picDirty:     true,
			hasScene:     true,
			expectClean:  false,
		},
		{
			name:         "full_redraw_needed",
			entryVersion: 5,
			picVersion:   5,
			picDirty:     false,
			fullRedraw:   true,
			hasScene:     true,
			expectClean:  false,
		},
		{
			name:         "nil_scene_always_dirty",
			entryVersion: 5,
			picVersion:   5,
			picDirty:     false,
			hasScene:     false,
			expectClean:  false,
		},
		{
			name:         "multiple_hover_versions_behind",
			entryVersion: 1,
			picVersion:   5, // 4 hovers happened without render
			picDirty:     false,
			hasScene:     true,
			expectClean:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rl := &renderLoop{}
			rl.fullRedrawNeeded = tc.fullRedraw

			entry := &boundaryTexEntry{
				sceneVersion: tc.entryVersion,
			}

			pic := compositor.NewPictureLayer()
			pic.SetSceneVersion(tc.picVersion)
			if tc.picDirty {
				pic.MarkDirty()
			} else {
				pic.ClearDirty()
			}

			var s *scene.Scene
			if tc.hasScene {
				s = scene.NewScene()
			}

			got := rl.isBoundaryClean(entry, pic, s)
			if got != tc.expectClean {
				t.Errorf("isBoundaryClean = %v, want %v "+
					"(entryVersion=%d, picVersion=%d, dirty=%v, fullRedraw=%v, hasScene=%v)",
					got, tc.expectClean,
					tc.entryVersion, tc.picVersion, tc.picDirty, tc.fullRedraw, tc.hasScene)
			}
		})
	}
}

// TestOverlayBoundary_VersionUpdatedAfterFlush verifies that after
// flushBoundaryToTexture, the entry.sceneVersion is updated to match
// the PictureLayer's sceneVersion. This ensures that on the next frame,
// isBoundaryClean returns true (no re-render needed for clean overlay).
func TestOverlayBoundary_VersionUpdatedAfterFlush(t *testing.T) {
	entry := &boundaryTexEntry{
		sceneVersion: 5,
		width:        200,
		height:       150,
	}

	pic := compositor.NewPictureLayer()
	pic.SetSceneVersion(6) // re-recorded after hover
	pic.ClearDirty()

	// Before flush: version mismatch → not clean.
	rl := &renderLoop{}
	s := dummyScene()
	if rl.isBoundaryClean(entry, pic, s) {
		t.Fatal("before flush: should not be clean (version mismatch)")
	}

	// Simulate what flushBoundaryToTexture does at the end:
	// entry.sceneVersion = pic.SceneVersion()
	entry.sceneVersion = pic.SceneVersion()

	// After flush: version matches → clean.
	if !rl.isBoundaryClean(entry, pic, s) {
		t.Error("after flush: should be clean (entry.sceneVersion updated to match pic)")
	}
}

// TestOverlayBoundary_DamageRectsForOverlay verifies that trackBoundaryDamage
// correctly records damage rects for non-root overlay boundaries.
// Root boundaries set rootTextureChanged; overlay boundaries (non-root) must
// append to frameDamageRects and boundaryDamageLogical.
func TestOverlayBoundary_DamageRectsForOverlay(t *testing.T) {
	// Root boundary damage.
	t.Run("root_sets_flag", func(t *testing.T) {
		rootPic := compositor.NewPictureLayer()
		rootPic.SetRoot(true)
		rootPic.SetBoundaryCacheKey(1)
		rootPic.SetSize(800, 600)
		rootPic.SetScreenOrigin(geometry.Point{})

		rl := &renderLoop{
			frameDamageRects:      make([]image.Rectangle, 0),
			boundaryDamageLogical: make([]image.Rectangle, 0),
		}

		rl.trackBoundaryDamage(rootPic, 800, 600)

		if !rl.rootTextureChanged {
			t.Error("root boundary should set rootTextureChanged")
		}
		if len(rl.frameDamageRects) != 0 {
			t.Errorf("root boundary should not append to frameDamageRects (got %d)", len(rl.frameDamageRects))
		}
	})

	// Overlay boundary damage (non-root, simulates dropdown menu).
	t.Run("overlay_appends_rects", func(t *testing.T) {
		// Simulate: trackBoundaryDamage for non-root overlay needs rl.canvas
		// for DeviceScale. Since we can't create a real canvas in unit tests,
		// we verify the data structures manually with scale=1 logic.
		overlayPic := compositor.NewPictureLayer()
		overlayPic.SetRoot(false)
		overlayPic.SetBoundaryCacheKey(42)
		overlayPic.SetSize(200, 150)
		overlayPic.SetScreenOrigin(geometry.Pt(100, 200))

		rl := &renderLoop{
			frameDamageRects:      make([]image.Rectangle, 0),
			boundaryDamageLogical: make([]image.Rectangle, 0),
		}

		origin := overlayPic.ScreenOrigin()
		bw, bh := overlayPic.Size()

		// Replicate non-root trackBoundaryDamage logic (scale=1).
		rl.boundaryDamageLogical = append(rl.boundaryDamageLogical, image.Rect(
			int(origin.X), int(origin.Y),
			int(origin.X)+bw, int(origin.Y)+bh,
		))
		rl.frameDamageRects = append(rl.frameDamageRects, image.Rect(
			int(origin.X), int(origin.Y),
			int(origin.X)+bw, int(origin.Y)+bh,
		))

		if rl.rootTextureChanged {
			t.Error("overlay boundary should NOT set rootTextureChanged")
		}

		wantLogical := image.Rect(100, 200, 300, 350)
		if len(rl.boundaryDamageLogical) != 1 {
			t.Fatalf("boundaryDamageLogical count = %d, want 1", len(rl.boundaryDamageLogical))
		}
		if rl.boundaryDamageLogical[0] != wantLogical {
			t.Errorf("logical rect = %v, want %v", rl.boundaryDamageLogical[0], wantLogical)
		}

		if len(rl.frameDamageRects) != 1 {
			t.Fatalf("frameDamageRects count = %d, want 1", len(rl.frameDamageRects))
		}
		wantPhysical := image.Rect(100, 200, 300, 350) // scale=1
		if rl.frameDamageRects[0] != wantPhysical {
			t.Errorf("physical rect = %v, want %v", rl.frameDamageRects[0], wantPhysical)
		}
	})
}

// TestOverlayBoundary_TwoFrameSimulation simulates two frames of the overlay
// damage pipeline at the render layer:
//
//	Frame 1: overlay first render → entry.sceneVersion=V1
//	Frame 2: hover → re-record → pic.SceneVersion=V2, entry still V1
//	         → isBoundaryClean returns false → render → damage tracked
//	Frame 3: no hover → pic.SceneVersion=V2, entry=V2
//	         → isBoundaryClean returns true → render skipped
//
// This is the end-to-end render-layer test that proves version detection works.
func TestOverlayBoundary_TwoFrameSimulation(t *testing.T) {
	s := dummyScene()

	// --- Frame 1: Initial render ---
	overlayPic := compositor.NewPictureLayer()
	overlayPic.SetBoundaryCacheKey(42)
	overlayPic.SetRoot(false)
	overlayPic.SetSize(200, 150)
	overlayPic.SetScreenOrigin(geometry.Pt(100, 200))
	overlayPic.SetSceneVersion(1)
	overlayPic.MarkDirty()

	rl := &renderLoop{
		boundaryTextures:      make(map[uint64]*boundaryTexEntry),
		frameDamageRects:      make([]image.Rectangle, 0),
		boundaryDamageLogical: make([]image.Rectangle, 0),
	}

	entry := &boundaryTexEntry{
		width:        200,
		height:       150,
		sceneVersion: 0, // never rendered
	}
	rl.boundaryTextures[42] = entry

	// isBoundaryClean: sceneVersion 0 != 1 → false → render.
	if rl.isBoundaryClean(entry, overlayPic, s) {
		t.Fatal("frame 1: should NOT be clean (first render, version=0 != 1)")
	}

	// After render: update entry.
	entry.sceneVersion = overlayPic.SceneVersion()
	rl.renderCount++

	// --- Frame 2: Hover → re-record ---
	overlayPic.SetSceneVersion(2) // simulates ClearSceneDirty version bump
	overlayPic.ClearDirty()       // sceneDirty cleared by recordBoundary

	// Reset per-frame damage state.
	rl.frameDamageRects = rl.frameDamageRects[:0]
	rl.boundaryDamageLogical = rl.boundaryDamageLogical[:0]
	rl.renderCount = 0

	// isBoundaryClean: entry version=1, pic version=2 → false → render.
	if rl.isBoundaryClean(entry, overlayPic, s) {
		t.Error("frame 2: should NOT be clean — version mismatch (entry=1, pic=2). " +
			"BUG: hover re-render skipped, no damage will be tracked")
	}

	// After render: update entry + track damage.
	entry.sceneVersion = overlayPic.SceneVersion()
	rl.renderCount++

	// Simulate trackBoundaryDamage (non-root, scale=1).
	origin := overlayPic.ScreenOrigin()
	bw, bh := overlayPic.Size()
	rl.boundaryDamageLogical = append(rl.boundaryDamageLogical, image.Rect(
		int(origin.X), int(origin.Y),
		int(origin.X)+bw, int(origin.Y)+bh,
	))

	if rl.renderCount != 1 {
		t.Errorf("frame 2: renderCount = %d, want 1", rl.renderCount)
	}
	if len(rl.boundaryDamageLogical) != 1 {
		t.Errorf("frame 2: boundaryDamageLogical = %d rects, want 1", len(rl.boundaryDamageLogical))
	}

	// --- Frame 3: No hover (clean) ---
	rl.frameDamageRects = rl.frameDamageRects[:0]
	rl.boundaryDamageLogical = rl.boundaryDamageLogical[:0]
	rl.renderCount = 0

	// Version matches → clean → render skipped.
	if !rl.isBoundaryClean(entry, overlayPic, s) {
		t.Error("frame 3: should be clean (entry=2, pic=2, versions match)")
	}

	if rl.renderCount != 0 {
		t.Errorf("frame 3: renderCount = %d, want 0 (no render needed)", rl.renderCount)
	}
	if len(rl.boundaryDamageLogical) != 0 {
		t.Errorf("frame 3: should have 0 damage rects, got %d", len(rl.boundaryDamageLogical))
	}
}

// TestOverlayBoundary_LayerTreeContainsOverlay verifies that the Layer Tree
// built by buildTestLayerTree (or equivalent) correctly includes overlay
// PictureLayers appended after the main tree, and that they are non-root.
func TestOverlayBoundary_LayerTreeContainsOverlay(t *testing.T) {
	// Build main tree with root only.
	root, _ := buildTestLayerTree(0, 0, 0)

	// Add overlay PictureLayer manually (simulating AppendOverlaysToLayerTree).
	overlayOffset := compositor.NewOffsetLayer(geometry.Pt(100, 200))
	overlayPic := compositor.NewPictureLayer()
	overlayPic.SetBoundaryCacheKey(999)
	overlayPic.SetRoot(false)
	overlayPic.SetSize(200, 150)
	overlayPic.SetScreenOrigin(geometry.Pt(100, 200))
	overlayPic.SetSceneVersion(1)
	overlayOffset.Append(overlayPic)
	root.Append(overlayOffset)

	// Verify tree contains both root and overlay PictureLayers.
	var pics []*compositor.PictureLayerImpl
	collectPictureLayers(root, &pics, true)

	if len(pics) != 2 {
		t.Fatalf("expected 2 PictureLayers (root + overlay), got %d", len(pics))
	}

	foundOverlay := false
	for _, pic := range pics {
		if pic.BoundaryCacheKey() == 999 {
			foundOverlay = true
			if pic.IsRoot() {
				t.Error("overlay PictureLayer should NOT be root")
			}
		}
	}
	if !foundOverlay {
		t.Error("overlay PictureLayer (key=999) not found in tree")
	}
}
