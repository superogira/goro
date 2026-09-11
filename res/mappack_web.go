//go:build js && wasm

package res

import (
	"bytes"
	"time"
)

const (
	webMapPackDir   = "data/map_pack/"
	webMapPackKeep  = 3
	webMapPackLimit = 60 * time.Second
)

// EnsureWebMapPack loads data/map_pack/<mapBase>.grf before the map's
// files are read, so every resource the map references resolves from one
// download instead of per-file HTTP round trips. A missing, failed, or
// partial pack is not an error: reads fall through to the loose files
// exactly as they did before. Must run on the game goroutine — it blocks
// on the download and then appends to Archives, which the frame loop
// reads without a lock.
func (m *Manager) EnsureWebMapPack(mapBase string) {
	if m == nil || mapBase == "" {
		return
	}
	if m.webMapPacks == nil {
		m.webMapPacks = &webMapPackState{
			loaded: make(map[string]*GRF),
			failed: make(map[string]struct{}),
		}
	}
	p := m.webMapPacks
	p.mu.Lock()
	if _, ok := p.failed[mapBase]; ok {
		p.mu.Unlock()
		return
	}
	if _, ok := p.loaded[mapBase]; ok {
		p.mu.Unlock()
		return
	}
	p.loaded[mapBase] = nil // in flight / done marker; archive set below
	p.mu.Unlock()

	name := webMapPackDir + mapBase + ".grf"
	data, err := webFetchPack(name, webMapPackLimit, func(loaded, total int64) {
		reportPackProgress(name, loaded, total)
	})
	var archive *GRF
	if err == nil {
		archive, err = OpenGRFReader(name, bytes.NewReader(data))
	}
	reportPackDone(name, err == nil)

	p.mu.Lock()
	defer p.mu.Unlock()
	if err != nil {
		delete(p.loaded, mapBase)
		p.failed[mapBase] = struct{}{}
		return
	}
	p.loaded[mapBase] = archive
	m.Archives = append(m.Archives, archive)
	for i, base := range p.order {
		if base == mapBase {
			p.order = append(p.order[:i], p.order[i+1:]...)
			break
		}
	}
	p.order = append(p.order, mapBase)
	for len(p.order) > webMapPackKeep {
		oldest := p.order[0]
		p.order = p.order[1:]
		if evicted := p.loaded[oldest]; evicted != nil {
			for i, archive := range m.Archives {
				if archive == evicted {
					m.Archives = append(m.Archives[:i], m.Archives[i+1:]...)
					break
				}
			}
		}
		delete(p.loaded, oldest)
		// The evicted map can come back later if the player returns; probe again then.
	}
}
