//go:build !js || !wasm

package res

import "time"

// PrefetchHandle is the native counterpart of the web prefetch handle.
// Native builds read from local disk/GRF, where a synchronous read is a few
// microseconds — background warming has nothing to win.
type PrefetchHandle struct{}

// Done always reports true on native: callers proceed straight to the
// synchronous load, exactly as they did before prefetching existed.
func (h *PrefetchHandle) Done() bool {
	return true
}

// Stalled always reports false on native.
func (h *PrefetchHandle) Stalled(grace time.Duration) bool {
	return false
}

// Prefetch is a no-op on native builds.
func (m *Manager) Prefetch(groups ...[]string) *PrefetchHandle {
	return &PrefetchHandle{}
}

// PrefetchTick is a no-op on native builds.
func PrefetchTick() {}
