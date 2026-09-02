package desktop

import "testing"

// TestScaleToPhysical pins the logical→physical conversion used to size
// per-boundary offscreen textures. Allocating these textures in logical pixels
// while the gg context renders at DeviceScale clips the scene and upscales it on
// composite — the "content rendered at 2x" symptom on HiDPI/Retina (#129).
func TestScaleToPhysical(t *testing.T) {
	tests := []struct {
		name         string
		w, h         int
		scale        float64
		wantW, wantH int
	}{
		{"1x is identity", 1100, 879, 1.0, 1100, 879},
		{"2x retina doubles", 1100, 879, 2.0, 2200, 1758},
		{"1.5x rounds to nearest", 100, 100, 1.5, 150, 150},
		{"fractional rounds up at .5", 3, 3, 1.5, 5, 5},
		{"zero scale treated as 1x", 640, 480, 0, 640, 480},
		{"negative scale treated as 1x", 640, 480, -2, 640, 480},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotW, gotH := scaleToPhysical(tt.w, tt.h, tt.scale)
			if gotW != tt.wantW || gotH != tt.wantH {
				t.Errorf("scaleToPhysical(%d, %d, %v) = (%d, %d), want (%d, %d)",
					tt.w, tt.h, tt.scale, gotW, gotH, tt.wantW, tt.wantH)
			}
		})
	}
}
