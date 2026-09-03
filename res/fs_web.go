//go:build js && wasm

package res

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"syscall/js"
	"time"
)

// webCacheBudget caps the in-memory resource cache. When the budget is
// exceeded the cache is dropped; browser HTTP caching (nginx sends
// Last-Modified) makes refetches cheap.
const webCacheBudget = 256 << 20

// webMissLimit caps the negative-cache of paths known to 404. Resource
// loaders probe several filename variants per lookup, so remembering misses
// keeps redundant requests down without growing forever.
const webMissLimit = 8192

type webFileCache struct {
	mu     sync.Mutex
	data   map[string][]byte
	misses map[string]struct{}
	bytes  int
}

var webCache = &webFileCache{
	data:   make(map[string][]byte),
	misses: make(map[string]struct{}),
}

func (c *webFileCache) get(path string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, ok := c.data[path]
	return data, ok
}

func (c *webFileCache) getMiss(path string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.misses[path]
	return ok
}

func (c *webFileCache) put(path string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.data[path]; exists {
		return
	}
	if c.bytes+len(data) > webCacheBudget {
		c.data = make(map[string][]byte)
		c.bytes = 0
	}
	c.data[path] = data
	c.bytes += len(data)
}

func (c *webFileCache) putMiss(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.misses) >= webMissLimit {
		c.misses = make(map[string]struct{})
	}
	c.misses[path] = struct{}{}
}

// webURLPath percent-encodes each path segment so resource names containing
// backslashes, spaces, or non-ASCII characters (Korean texture folders)
// resolve to the same URLs the static server serves.
func webURLPath(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}

// webPathDialect remembers which URL dialect this host accepted, learned
// from the first successful load. Static hosts consistently speak one
// dialect (forward slashes, UTF-8 names, one casing), so remembering it
// collapses steady-state probing to a single request per file instead of
// burning 404 round trips on separator, charset, and case variants that
// never work on this server.
type webPathDialect struct {
	slash bool // forward-slash form beat the backslash original
	utf8  bool // the EUC-KR-decoded (UTF-8) variant beat the raw bytes
	lower bool // lowercased spelling won
	upper bool // uppercased spelling won
}

var webDialect struct {
	sync.Mutex
	learned  bool
	settings webPathDialect
}

// noteCandidateOutcome learns the dialect from a candidate that downloaded
// successfully.
func noteCandidateOutcome(c webCandidate) {
	webDialect.Lock()
	defer webDialect.Unlock()
	if webDialect.learned {
		return // keep the first win; mixed dialects are rare and the
		// full candidate list is still probed on misses
	}
	webDialect.learned = true
	webDialect.settings = webPathDialect{
		slash: c.slash,
		utf8:  c.utf8,
		lower: c.lower,
		upper: c.upper,
	}
}

// candidateTags remembers the dialect traits of recently built probe URLs
// so download success can feed the learned dialect. Bounded; probe URLs are
// short-lived (per-resource) and cleared wholesale once over budget.
var candidateTags = struct {
	sync.Mutex
	m map[string]webCandidate
}{m: make(map[string]webCandidate)}

func rememberCandidate(c webCandidate) {
	candidateTags.Lock()
	defer candidateTags.Unlock()
	if len(candidateTags.m) > 4096 {
		candidateTags.m = make(map[string]webCandidate)
	}
	candidateTags.m[c.url] = c
}

func learnFromSuccess(url string) {
	candidateTags.Lock()
	c, ok := candidateTags.m[url]
	candidateTags.Unlock()
	if ok {
		noteCandidateOutcome(c)
	}
}

// webCandidate is one probe URL tagged with the dialect traits it carries.
type webCandidate struct {
	url   string
	slash bool
	utf8  bool
	lower bool
	upper bool
}

// candidatePaths returns the URLs to probe for a normalized resource name,
// rooted at the page origin, ordered most-likely-first:
//   - the learned dialect's traits first (after the first successful load),
//   - then UTF-8 forward-slash spellings (what static web hosts serve),
//   - then the raw separator/charset spellings,
//   - case variants last (kRO data references are case-inconsistent; Linux
//     servers 404 the wrong casing, which surfaces as white map tiles).
func (m *Manager) candidatePaths(normalized string) []string {
	base := strings.TrimSuffix(m.Root, "/")
	seen := make(map[string]struct{}, 4)
	var candidates []webCandidate
	add := func(path string, slash, utf8, lower, upper bool) {
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		c := webCandidate{
			url:   base + "/" + webURLPath(path),
			slash: slash,
			utf8:  utf8,
			lower: lower,
			upper: upper,
		}
		rememberCandidate(c)
		candidates = append(candidates, c)
	}

	raw := normalized
	rawSlash := strings.ReplaceAll(normalized, "\\", "/")
	// euckrVariant yields the UTF-8 spelling when the raw name carries
	// CP949 bytes from GRF-era map files; for already-valid UTF-8 names it
	// is empty and the original doubles as the UTF-8 form.
	utf8Form := euckrVariant(raw)
	if utf8Form == "" {
		utf8Form = raw
	}
	utf8Slash := strings.ReplaceAll(utf8Form, "\\", "/")

	add(utf8Slash, true, true, false, false)
	add(utf8Form, false, true, false, false)
	add(rawSlash, true, false, false, false)
	add(raw, false, false, false, false)
	for _, variant := range []struct {
		path  string
		lower bool
	}{
		{strings.ToLower(utf8Slash), true},
		{strings.ToUpper(utf8Slash), false},
		{strings.ToLower(utf8Form), true},
		{strings.ToUpper(utf8Form), false},
		{strings.ToLower(rawSlash), true},
		{strings.ToUpper(rawSlash), false},
	} {
		upper := !variant.lower
		add(variant.path, strings.Contains(variant.path, "/"), variant.path != raw && variant.path != rawSlash || utf8Form != raw, variant.lower, upper)
	}

	// Order by agreement with the learned dialect (stable for ties).
	webDialect.Lock()
	d := webDialect.settings
	learned := webDialect.learned
	webDialect.Unlock()
	if learned {
		score := func(c webCandidate) int {
			n := 0
			if c.slash == d.slash {
				n++
			}
			if c.utf8 == d.utf8 {
				n++
			}
			if d.lower && c.lower {
				n++
			} else if d.upper && c.upper {
				n++
			}
			return n
		}
		sort.SliceStable(candidates, func(i, j int) bool {
			return score(candidates[i]) > score(candidates[j])
		})
	}
	out := make([]string, len(candidates))
	for i, c := range candidates {
		out[i] = c.url
	}
	return out
}

// candidateExists reports whether a candidate URL downloads successfully.
// The body is cached so the follow-up readCandidate does not refetch.
func (m *Manager) candidateExists(candidate string) bool {
	if _, ok := webCache.get(candidate); ok {
		return true
	}
	if webCache.getMiss(candidate) {
		return false
	}
	data, err := webFetch(candidate)
	if err != nil {
		webCache.putMiss(candidate)
		return false
	}
	learnFromSuccess(candidate)
	webCache.put(candidate, data)
	return true
}

// readCandidate returns the contents of a candidate URL returned by Find.
func (m *Manager) readCandidate(candidate string) ([]byte, error) {
	if data, ok := webCache.get(candidate); ok {
		return data, nil
	}
	if webCache.getMiss(candidate) {
		return nil, fmt.Errorf("resource not found: %s", candidate)
	}
	data, err := webFetch(candidate)
	if err != nil {
		webCache.putMiss(candidate)
		return nil, err
	}
	webCache.put(candidate, data)
	return data, nil
}

// scanArchives is a no-op on the web build: data is served as loose files.
func (m *Manager) scanArchives() {}

// webFetch downloads a same-origin URL with the Fetch API. The promise is
// bridged onto a channel; instead of parking on the channel forever, the
// caller polls with a timer sleep so the wasm runtime keeps servicing the
// JS event loop that settles the fetch promise.
func webFetch(path string) ([]byte, error) {
	type result struct {
		data []byte
		err  error
	}
	done := make(chan result, 1)

	onRejected := js.FuncOf(func(this js.Value, args []js.Value) any {
		err := fmt.Errorf("fetch %s failed", path)
		if len(args) > 0 {
			reason := args[0]
			if reason.Type() == js.TypeString {
				err = fmt.Errorf("fetch %s: %s", path, reason.String())
			}
		}
		done <- result{err: err}
		return nil
	})
	onFulfilled := js.FuncOf(func(this js.Value, args []js.Value) any {
		response := args[0]
		if !response.Get("ok").Bool() {
			done <- result{err: fmt.Errorf("fetch %s: HTTP %s", path, response.Get("status").String())}
			return nil
		}
		bufferPromise := response.Call("arrayBuffer")
		bufferPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			buffer := args[0]
			size := buffer.Get("byteLength").Int()
			data := make([]byte, size)
			js.CopyBytesToGo(data, js.Global().Get("Uint8Array").New(buffer))
			done <- result{data: data}
			return nil
		}), onRejected)
		return nil
	})

	js.Global().Call("fetch", path).Call("then", onFulfilled, onRejected)
	defer onFulfilled.Release()
	defer onRejected.Release()

	deadline := time.Now().Add(60 * time.Second)
	for {
		select {
		case out := <-done:
			return out.data, out.err
		default:
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("fetch %s: timed out", path)
		}
		time.Sleep(2 * time.Millisecond)
	}
}
