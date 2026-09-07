//go:build js && wasm

package res

import (
	"sync"
	"syscall/js"
	"time"
)

// PrefetchHandle tracks a cache-warming job. Done reports whether every
// requested file has been fetched (or definitively missed), so a subsequent
// synchronous ReadFile is served from the web cache with zero network round
// trips. Stalled reports jobs that stopped making progress so callers can
// fall back to the old synchronous load instead of dropping the sprite.
type PrefetchHandle struct {
	started time.Time
	done    bool
}

// Done reports whether the prefetch job finished. A nil handle is always
// done, letting callers treat "no prefetch started" as ready.
func (h *PrefetchHandle) Done() bool {
	return h == nil || h.done
}

// Stalled reports whether the job is unfinished and older than grace.
func (h *PrefetchHandle) Stalled(grace time.Duration) bool {
	return h != nil && !h.done && time.Since(h.started) > grace
}

// prefetchFetches tracks in-flight URL fetches: entry with a nil result is
// pending, non-nil is settled. JS fetch callbacks write here from their own
// activations, so the map is mutex-guarded; everything else in this file
// runs on the game goroutine via PrefetchTick.
var prefetchFetches = struct {
	sync.Mutex
	m map[string]*prefetchFetch
}{m: make(map[string]*prefetchFetch)}

type prefetchFetch struct {
	launched time.Time
	result   *prefetchResult
}

type prefetchResult struct {
	ok   bool
	data []byte
}

// prefetchInFlightCap bounds concurrent URL fetches. The goal is to overlap
// round trips with frame time, not to saturate the browser's connections.
const prefetchInFlightCap = 6

// prefetchFetchTimeout drops fetches that never settle so their job can move
// on (the synchronous loader will retry and surface real errors).
const prefetchFetchTimeout = 30 * time.Second

type prefetchJob struct {
	manager    *Manager
	groups     [][]string
	groupIndex int
	nameIndex  int
	urls       []string // URL candidates for groups[groupIndex][nameIndex]
	urlIndex   int
	// inFlightLaunched is the job-local clock for the in-flight URL; the
	// shared entry can be deleted by another consumer at any time.
	inFlightLaunched time.Time
	inFlight         string
	handle           *PrefetchHandle
}

// prefetchQueue holds interactive jobs (sprites, sounds); backgroundQueue
// holds bulk warming (map textures). Both are only touched from the game
// goroutine (Prefetch appends, PrefetchTick advances).
var (
	prefetchQueue   []*prefetchJob
	prefetchBgQueue []*prefetchJob
)

// Prefetch warms the web file cache for the first existing candidate of each
// group, driven a step at a time from PrefetchTick instead of a goroutine:
// the wasm build is effectively single-goroutine, and a background worker
// polling with sleeps can starve the frame loop. Groups mirror the candidate
// lists the matching loader passes to readFirstResource; the loader remains
// the source of truth, so a wrong or missing entry only costs one request.
// Interactive callers get foreground scheduling: a large map-texture batch
// must never delay a sprite the player is waiting to see.
func (m *Manager) Prefetch(groups ...[]string) *PrefetchHandle {
	return m.enqueuePrefetch(groups, &prefetchQueue)
}

// PrefetchBackground schedules bulk cache warming that yields to every
// foreground job (map ground/model/water textures can be hundreds of
// files; leaving them in the same FIFO starved sprite pop-in for seconds).
func (m *Manager) PrefetchBackground(groups ...[]string) *PrefetchHandle {
	return m.enqueuePrefetch(groups, &prefetchBgQueue)
}

func (m *Manager) enqueuePrefetch(groups [][]string, queue *[]*prefetchJob) *PrefetchHandle {
	handle := &PrefetchHandle{started: time.Now()}
	if len(prefetchQueue)+len(prefetchBgQueue) >= 512 {
		handle.done = true
		return handle
	}
	*queue = append(*queue, &prefetchJob{
		manager: m,
		groups:  groups,
		handle:  handle,
	})
	return handle
}

// PrefetchTick advances every prefetch job by at most one step: it launches
// the next URL fetch when a slot is free, or folds a settled fetch into the
// cache and moves the cursors. Called once per frame from the game update.
func PrefetchTick() {
	for i := 0; i < len(prefetchQueue); i++ {
		if prefetchStep(prefetchQueue[i]) {
			// The handle's Done() gates render-time fallbacks; without this
			// every view waited out the full stall grace even though its
			// files were already warm.
			prefetchQueue[i].handle.done = true
			prefetchQueue = append(prefetchQueue[:i], prefetchQueue[i+1:]...)
			i--
		}
	}
	// Background jobs only advance when no foreground job is waiting.
	if len(prefetchQueue) == 0 && len(prefetchBgQueue) > 0 {
		for i := 0; i < len(prefetchBgQueue); i++ {
			if prefetchStep(prefetchBgQueue[i]) {
				prefetchBgQueue[i].handle.done = true
				prefetchBgQueue = append(prefetchBgQueue[:i], prefetchBgQueue[i+1:]...)
				i--
			}
		}
	}
}

// advanceGroup consumes the current group and reports whether the whole
// job is finished.
func (job *prefetchJob) advanceGroup() bool {
	job.groupIndex++
	job.nameIndex, job.urlIndex, job.urls = 0, 0, nil
	return job.groupIndex >= len(job.groups)
}

// prefetchStep advances one job; it reports true when the job is finished.
func prefetchStep(job *prefetchJob) bool {
	if job.inFlight != "" {
		settled, ok := prefetchSettled(job.inFlight)
		if !ok {
			// Either genuinely pending, or the entry was consumed by
			// another job waiting on the same URL (its result already
			// landed in the web cache). Distinguish by existence; the
			// timeout uses the job-local launch clock because a deleted
			// entry has no durable one.
			if _, exists := prefetchGet(job.inFlight); exists {
				if time.Since(job.inFlightLaunched) <= prefetchFetchTimeout {
					return false
				}
				prefetchForget(job.inFlight)
				webCache.putMiss(job.inFlight)
			}
			job.inFlight = ""
		} else {
			prefetchForget(job.inFlight)
			url := job.inFlight
			job.inFlight = ""
			if settled.ok {
				learnFromSuccess(url)
				webCache.put(url, settled.data)
				// First existing candidate is the one the loader uses;
				// the group is satisfied (Find/readFirstResource semantics).
				if job.advanceGroup() {
					return true
				}
			} else {
				webCache.putMiss(url)
			}
			// Fall through to try the next URL (or name, or group).
		}
	}
	if prefetchInFlightCount() >= prefetchInFlightCap {
		// Self-heal: entries whose owning job is gone would hold the cap
		// forever. Expire them by launch age.
		prefetchPurgeOlderThan(prefetchFetchTimeout)
		if prefetchInFlightCount() >= prefetchInFlightCap {
			return false
		}
	}
groups:
	for job.groupIndex < len(job.groups) {
		group := job.groups[job.groupIndex]
		if job.urls == nil && job.nameIndex < len(group) {
			// Whole-group archive pre-scan: when any spelling of this
			// sound/sprite lives in the web pack, the group is satisfied
			// without probing the wrong spellings that would 404 first.
			for _, name := range group {
				for _, u := range job.manager.candidatePaths(normalizePath(name)) {
					if job.manager.archiveHasCandidate(u) {
						if job.advanceGroup() {
							return true
						}
						continue groups
					}
				}
			}
			job.urls = job.manager.candidatePaths(normalizePath(group[job.nameIndex]))
			job.urlIndex = 0
		}
		for job.urlIndex < len(job.urls) {
			url := job.urls[job.urlIndex]
			job.urlIndex++
			if webCache.getMiss(url) {
				continue
			}
			if _, ok := webCache.get(url); ok {
				// Already cached: the group resolves without a fetch.
				if job.advanceGroup() {
					return true
				}
				continue groups
			}
			if job.manager.archiveHasCandidate(url) {
				// Packed in data_web.grf: the real read will come from
				// memory — no network fetch to warm.
				if job.advanceGroup() {
					return true
				}
				continue groups
			}
			job.inFlightLaunched = time.Now()
			prefetchLaunch(url)
			job.inFlight = url
			return false
		}
		// URL candidates for this name are exhausted (all missed): try the
		// next name in the group.
		job.nameIndex++
		job.urls, job.urlIndex = nil, 0
		if job.nameIndex >= len(group) {
			job.groupIndex++
			job.nameIndex = 0
		}
	}
	return true
}

func prefetchInFlightCount() int {
	prefetchFetches.Lock()
	defer prefetchFetches.Unlock()
	n := 0
	for _, f := range prefetchFetches.m {
		if f.result == nil {
			n++
		}
	}
	return n
}

func prefetchGet(url string) (*prefetchFetch, bool) {
	prefetchFetches.Lock()
	defer prefetchFetches.Unlock()
	f, ok := prefetchFetches.m[url]
	return f, ok
}

// prefetchPurgeOlderThan drops unsettled entries older than max. Entries
// whose owning job finished through another path (a shared URL consumed by
// a different job) would otherwise hold the in-flight cap forever.
func prefetchPurgeOlderThan(max time.Duration) {
	prefetchFetches.Lock()
	defer prefetchFetches.Unlock()
	now := time.Now()
	for url, f := range prefetchFetches.m {
		if f.result == nil && now.Sub(f.launched) > max {
			delete(prefetchFetches.m, url)
		}
	}
}

func prefetchSettled(url string) (prefetchResult, bool) {
	prefetchFetches.Lock()
	defer prefetchFetches.Unlock()
	f, ok := prefetchFetches.m[url]
	if !ok || f.result == nil {
		return prefetchResult{}, false
	}
	return *f.result, true
}

func prefetchLaunched(url string) time.Time {
	prefetchFetches.Lock()
	defer prefetchFetches.Unlock()
	if f, ok := prefetchFetches.m[url]; ok {
		return f.launched
	}
	return time.Now()
}

func prefetchForget(url string) {
	prefetchFetches.Lock()
	defer prefetchFetches.Unlock()
	delete(prefetchFetches.m, url)
}

// prefetchLaunch fires a Fetch API request without blocking. Settlement is
// recorded by the promise callbacks (their own JS activations), and a later
// PrefetchTick folds it into the cache.
func prefetchLaunch(url string) {
	prefetchFetches.Lock()
	if _, exists := prefetchFetches.m[url]; exists {
		prefetchFetches.Unlock()
		return
	}
	prefetchFetches.m[url] = &prefetchFetch{launched: time.Now()}
	prefetchFetches.Unlock()

	settle := func(ok bool, data []byte) {
		prefetchFetches.Lock()
		defer prefetchFetches.Unlock()
		if f, exists := prefetchFetches.m[url]; exists && f.result == nil {
			f.result = &prefetchResult{ok: ok, data: data}
		}
	}
	onRejected := js.FuncOf(func(this js.Value, args []js.Value) any {
		settle(false, nil)
		return nil
	})
	onFulfilled := js.FuncOf(func(this js.Value, args []js.Value) any {
		response := args[0]
		if !response.Get("ok").Bool() {
			settle(false, nil)
			return nil
		}
		bufferPromise := response.Call("arrayBuffer")
		bufferPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			buffer := args[0]
			size := buffer.Get("byteLength").Int()
			data := make([]byte, size)
			js.CopyBytesToGo(data, js.Global().Get("Uint8Array").New(buffer))
			settle(true, data)
			return nil
		}), onRejected)
		return nil
	})
	js.Global().Call("fetch", url).Call("then", onFulfilled, onRejected)
}
