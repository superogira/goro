//go:build !js || !wasm

package res

// EnsureWebMapPack is a no-op on native builds: map packs are a web
// streaming optimization and the loose data directory serves reads directly.
func (m *Manager) EnsureWebMapPack(mapBase string) {}
