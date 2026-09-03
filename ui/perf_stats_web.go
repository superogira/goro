//go:build js && wasm

package ui

import (
	"runtime"
	"syscall/js"
)

// perfMemStats gathers memory figures for the perf HUD. On the browser the
// JS heap is reported alongside the Go heap: WASM modules get killed by the
// browser for either growing too far.
func perfMemStats() (m perfMem) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	m.goHeapMB = float64(ms.HeapAlloc) / 1024 / 1024
	m.goSysMB = float64(ms.Sys) / 1024 / 1024
	m.goGC = ms.NumGC

	pm := js.Global().Get("performance").Get("memory")
	if !pm.IsUndefined() {
		used := float64(pm.Get("usedJSHeapSize").Int()) / 1024 / 1024
		total := float64(pm.Get("totalJSHeapSize").Int()) / 1024 / 1024
		limit := float64(pm.Get("jsHeapSizeLimit").Int()) / 1024 / 1024
		m.extra = "JS " + formatMB(used) + "/" + formatMB(total) + "MB (limit " + formatMB(limit) + ")"
	} else {
		dm := js.Global().Get("navigator").Get("deviceMemory")
		if !dm.IsUndefined() {
			m.extra = "JS n/a, device RAM ~" + dm.String() + "GB"
		}
	}
	return m
}
