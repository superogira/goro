//go:build !js || !wasm

package ui

import "runtime"

func perfMemStats() (m perfMem) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	m.goHeapMB = float64(ms.HeapAlloc) / 1024 / 1024
	m.goSysMB = float64(ms.Sys) / 1024 / 1024
	m.goGC = ms.NumGC
	return m
}
