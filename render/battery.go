package render

import (
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// The handheld battery badge. The RG35XX (H700) exposes the battery the
// standard AXP sysfs way — a power_supply entry (named "battery" on this
// unit) carrying `capacity` (0-100) and `status` ("Charging" /
// "Discharging" / "Full"). ROCreader on the same hardware discovers it by
// scanning /sys/class/power_supply for a readable capacity file, preferring
// the entry literally named "battery"; this mirrors that scheme.

const (
	batterySysfsRoot    = "/sys/class/power_supply"
	batteryPollInterval = 5 * time.Second
)

type batteryState struct {
	percent  int
	charging bool
	ok       bool
}

var batteryNow atomic.Pointer[batteryState]

type batteryPaths struct {
	capacity string
	status   string
}

// discoverBatteryPaths scans the sysfs root the way ROCreader does: prefer
// the entry named "battery", else the first entry with a readable capacity
// file. The root is a parameter so tests can point it at a fake tree.
func discoverBatteryPaths(root string) (batteryPaths, bool) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return batteryPaths{}, false
	}
	first := ""
	for _, entry := range entries {
		name := entry.Name()
		if first == "" {
			if _, err := os.Stat(filepath.Join(root, name, "capacity")); err == nil {
				first = name
			}
		}
		if name == "battery" {
			if _, err := os.Stat(filepath.Join(root, name, "capacity")); err == nil {
				first = name
				break
			}
		}
	}
	if first == "" {
		return batteryPaths{}, false
	}
	return batteryPaths{
		capacity: filepath.Join(root, first, "capacity"),
		status:   filepath.Join(root, first, "status"),
	}, true
}

// readBattery reads one sample. Missing or malformed values leave ok=false
// so the badge hides rather than showing garbage.
func readBattery(paths batteryPaths) batteryState {
	data, err := os.ReadFile(paths.capacity)
	if err != nil {
		return batteryState{}
	}
	percent, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || percent < 0 || percent > 100 {
		return batteryState{}
	}
	charging := false
	if status, err := os.ReadFile(paths.status); err == nil {
		charging = strings.HasPrefix(strings.TrimSpace(string(status)), "Charg")
	}
	return batteryState{percent: percent, charging: charging, ok: true}
}

var batteryMonitorOnce sync.Once

func startBatteryMonitor() {
	go func() {
		t := time.NewTicker(batteryPollInterval)
		defer t.Stop()
		for range t.C {
			if paths, ok := discoverBatteryPaths(batterySysfsRoot); ok {
				batteryNow.Store(ptrOf(readBattery(paths)))
			} else {
				batteryNow.Store(&batteryState{})
			}
		}
	}()
	// Seed immediately so the badge is there on the first frame.
	if paths, ok := discoverBatteryPaths(batterySysfsRoot); ok {
		batteryNow.Store(ptrOf(readBattery(paths)))
	}
}

func ptrOf[T any](v T) *T { return &v }

// drawBatteryBadge renders the battery icon and percentage centered at the
// top edge on a dark panel — over bright maps the bare glyph was unreadable
// — using direct rects and direct-raster outlined text, the fbdev-proven
// paths (the queued-label text pipeline draws nothing there). While the
// on-screen keyboard is up it stands down: the OSK panel owns the
// top-center band.
func (r *runner) drawBatteryBadge(screen *Frame) {
	if screen == nil || OSKActive() {
		return
	}
	batteryMonitorOnce.Do(startBatteryMonitor)
	state := batteryNow.Load()
	if state == nil || !state.ok {
		return
	}
	bounds := screen.Bounds()
	const (
		bodyW = 22
		bodyH = 11
		tipW  = 2
		gap   = 6
		pad   = 5
	)
	percentText := strconv.Itoa(state.percent) + "%"
	textW := int(MeasureUIText(percentText, 12)) + 2
	const panelH = 18
	panelW := bodyW + tipW + gap + textW + pad*2
	x := (float64(bounds.Dx()) - float64(panelW)) / 2
	y := 4.0

	// Backdrop panel — background only, no border, matching the other
	// handheld overlays.
	DrawRect(screen, x, y, float64(panelW), panelH, color.RGBA{R: 14, G: 18, B: 24, A: 175})

	shell := color.RGBA{R: 235, G: 238, B: 244, A: 230}
	outline := color.RGBA{A: 190}
	var fill color.RGBA
	switch {
	case state.charging:
		fill = color.RGBA{R: 214, G: 178, B: 92, A: 255}
	case state.percent <= 15:
		fill = color.RGBA{R: 214, G: 84, B: 74, A: 255}
	case state.percent <= 40:
		fill = color.RGBA{R: 214, G: 178, B: 92, A: 255}
	default:
		fill = color.RGBA{R: 116, G: 200, B: 118, A: 255}
	}

	// Icon and text each vertically centered in the 18px panel: the 11px
	// body sits at +3, the 12px text at +2.
	ix := x + pad
	iy := y + 3
	// Body outline, tip on the right, proportional fill inside.
	DrawRect(screen, ix, iy, bodyW, 1, shell)
	DrawRect(screen, ix, iy+bodyH-1, bodyW, 1, shell)
	DrawRect(screen, ix, iy, 1, bodyH, shell)
	DrawRect(screen, ix+bodyW-1, iy, 1, bodyH, shell)
	DrawRect(screen, ix+bodyW, iy+3, tipW, bodyH-6, shell)
	innerW := bodyW - 4
	fillW := int(float64(innerW) * float64(state.percent) / 100)
	if state.percent > 0 && fillW < 1 {
		fillW = 1
	}
	DrawRect(screen, ix+2, iy+2, float64(fillW), bodyH-4, fill)

	// Charging: a small lightning bolt across the body.
	if state.charging {
		bolt := color.RGBA{R: 24, G: 20, B: 34, A: 255}
		cx := ix + bodyW/2
		DrawRect(screen, cx, iy+2, 2, 3, bolt)
		DrawRect(screen, cx-2, iy+5, 2, 3, bolt)
	}

	DrawOutlinedTextAt(screen, percentText, int(ix+bodyW+tipW+gap), int(y)+2, shell, outline)
}
