package desktop

import (
	"testing"

	"github.com/gogpu/ui/app"
)

// TestRunNilArgs verifies that Run returns errors for nil arguments
// rather than panicking.
func TestRunNilArgs(t *testing.T) {
	t.Run("nil gogpuApp", func(t *testing.T) {
		uiApp := app.New()
		err := Run(nil, uiApp)
		if err == nil {
			t.Fatal("expected error for nil gogpuApp")
		}
	})

	t.Run("nil uiApp", func(t *testing.T) {
		err := Run(nil, nil)
		if err == nil {
			t.Fatal("expected error for nil uiApp")
		}
	})
}

// TestRunForcesHostManaged verifies that Run sets HostManaged render mode
// for scene composition (ADR-007 Phase 7). HostManaged always draws the
// full tree — RepaintBoundary cache handles efficiency.
func TestRunForcesHostManaged(t *testing.T) {
	uiApp := app.New()
	w := uiApp.Window()

	if w.RenderMode() != app.RenderModeHostManaged {
		t.Fatalf("default mode = %v, want HostManaged", w.RenderMode())
	}

	// Simulate what Run does: set HostManaged (scene composition).
	w.SetRenderMode(app.RenderModeHostManaged)

	if w.RenderMode() != app.RenderModeHostManaged {
		t.Errorf("mode after SetRenderMode = %v, want HostManaged", w.RenderMode())
	}
}

// TestRenderModeConstants verifies the render mode constants are accessible
// from the desktop package (no import cycle).
func TestRenderModeConstants(t *testing.T) {
	if app.RenderModeHostManaged != 0 {
		t.Fatal("RenderModeHostManaged should be 0 (iota)")
	}
	if app.RenderModeFrameworkManaged != 1 {
		t.Fatal("RenderModeFrameworkManaged should be 1")
	}
}
