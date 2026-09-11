package res

import "sync"

// webMapPackState tracks per-map packs: which maps have a pack loaded or
// were probed and have none, plus the LRU order used to evict old packs.
type webMapPackState struct {
	mu     sync.Mutex
	loaded map[string]*GRF
	failed map[string]struct{}
	order  []string // least recently used first
}
