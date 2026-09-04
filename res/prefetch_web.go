//go:build js && wasm

package res

import (
	"syscall/js"
	"sync"
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
	inFlight   string
	handle     *PrefetchHandle
}

// prefetchQueue is only touched from the game goroutine (Prefetch appends,
// PrefetchTick advances).
var prefetchQueue []*prefetchJob

// Prefetch warms the web file cache for the first existing candidate of each
// group, driven a step at a time from PrefetchTick instead of a goroutine:
// the wasm build is effectively single-goroutine, and a background worker
// polling with sleeps can starve the frame loop. Groups mirror the candidate
// lists the matching loader passes to readFirstResource; the loader remains
// the source of truth, so a wrong or missing entry only costs one request.
func (m *Manager) Prefetch(groups ...[]string) *PrefetchHandle {
	handle := &PrefetchHandle{started: time.Now()}
	if len(prefetchQueue) >= 512 {
		handle.done = true
		return handle
	}
	prefetchQueue = append(prefetchQueue, &prefetchJob{
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
			prefetchQueue = append(prefetchQueue[:i], prefetchQueue[i+1:]...)
			i--
		}
	}
}

// prefetchStep advances one job; it reports true when the job is finished.
func prefetchStep(job *prefetchJob) bool {
	if job.inFlight != "" {
		settled, ok := prefetchSettled(job.inFlight)
		if !ok {
			if time.Since(prefetchLaunched(job.inFlight)) > prefetchFetchTimeout {
				prefetchForget(job.inFlight)
				webCache.putMiss(job.inFlight)
				job.inFlight = ""
			}
			return false
		}
		prefetchForget(job.inFlight)
		url := job.inFlight
		job.inFlight = ""
		if settled.ok {
			learnFromSuccess(url)
			webCache.put(url, settled.data)
			// First existing candidate is the one the loader uses; the
			// group is satisfied (same semantics as Find/readFirstResource).
			job.groupIndex++
			job.nameIndex, job.urlIndex, job.urls = 0, 0, nil
			return job.groupIndex >= len(job.groups)
		}
		webCache.putMiss(url)
		// Fall through to try the next URL (or name, or group).
	}
	if prefetchInFlightCount() >= prefetchInFlightCap {
		return false
	}
	for job.groupIndex < len(job.groups) {
		group := job.groups[job.groupIndex]
		if job.urls == nil && job.nameIndex < len(group) {
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
				job.groupIndex++
				job.nameIndex, job.urlIndex, job.urls = 0, 0, nil
				return job.groupIndex >= len(job.groups)
			}
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
