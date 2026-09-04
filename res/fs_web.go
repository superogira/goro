//go:build js && wasm

package res

import (
	"encoding/json"
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
	prefix      string // leading path segment that wins ("data/", "bgm/")
	slashWins   int    // forward-slash spellings that downloaded
	utf8Wins    int    // UTF-8 spellings that downloaded
	lowerWins   int    // lowercase spellings that downloaded
	exactWins   int    // original-case spellings that downloaded
	sampleCount int
}

var webDialect struct {
	sync.Mutex
	d webPathDialect
}

// noteCandidateOutcome folds a successful download into the dialect stats.
// Every win counts (not just the first): hosts consistently speak one
// dialect, so the aggregates converge to it within a couple of loads.
func noteCandidateOutcome(c webCandidate) {
	loadDialectOnce()
	webDialect.Lock()
	d := &webDialect.d
	if c.slash {
		d.slashWins++
	}
	if c.utf8 {
		d.utf8Wins++
	}
	if c.lower {
		d.lowerWins++
	} else if !c.upper {
		d.exactWins++
	}
	if d.prefix == "" && c.path != "" {
		// Remember the leading segment of the first success ("data/",
		// "bgm/"); hosts keep all resources under one root.
		p := strings.ReplaceAll(c.path, "\\", "/")
		if i := strings.Index(p, "/"); i > 0 {
			d.prefix = p[:i+1]
		}
	}
	d.sampleCount++
	webDialect.Unlock()
	saveDialect()
}

// candidateScore ranks how well a candidate matches the learned dialect.
func candidateScore(c webCandidate) int {
	webDialect.Lock()
	d := webDialect.d
	webDialect.Unlock()
	if d.sampleCount == 0 {
		return 0
	}
	score := 0
	if d.prefix != "" {
		p := strings.ReplaceAll(c.path, "\\", "/")
		if strings.HasPrefix(strings.ToLower(p), d.prefix) {
			score += 4 // this host keeps everything under the learned root
		}
	}
	if d.slashWins > 0 && c.slash {
		score++
	}
	if d.utf8Wins > 0 && c.utf8 {
		score++
	}
	if d.lowerWins > d.exactWins && c.lower {
		score += 2 // lowercase filenames dominate on this host
	}
	return score
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
	if !ok {
		return
	}
	noteCandidateOutcome(c)
	if c.path != "" {
		learnDirCase(c.path)
	}
}

// webCandidate is one probe URL tagged with the dialect traits it carries.
type webCandidate struct {
	url   string
	path  string // decoded path (for directory-case learning)
	slash bool
	utf8  bool
	lower bool
	upper bool
}

// webDirCase remembers the exact casing of each directory that served a
// file successfully. Hosts extract kRO data with inconsistent per-folder
// casing (BGM uppercase, npc/texture lowercase), so a single global
// dialect cannot capture it; rewriting directory prefixes to their known
// casing removes the per-directory 404 probes entirely.
var webDirCase = struct {
	sync.Mutex
	dirs map[string]string // lowercase dir path -> actual casing that worked
}{dirs: make(map[string]string)}

// dialectStoreKey is the localStorage key for the learned host dialect, so
// the probe ordering survives page reloads — without it every session paid
// the discovery 404s again on its first few files (title BGM included).
const dialectStoreKey = "goroPathDialect"

var dialectLoadOnce sync.Once

type dialectPersist struct {
	Prefix    string            `json:"prefix"`
	SlashWins int               `json:"slash"`
	Utf8Wins  int               `json:"utf8"`
	LowerWins int               `json:"lower"`
	ExactWins int               `json:"exact"`
	Samples   int               `json:"n"`
	DirCase   map[string]string `json:"dirs"`
}

func loadDialectOnce() {
	dialectLoadOnce.Do(func() {
		storage := js.Global().Get("localStorage")
		if storage.IsUndefined() {
			return
		}
		raw := storage.Call("getItem", dialectStoreKey)
		if raw.Type() != js.TypeString {
			return
		}
		var saved dialectPersist
		if err := json.Unmarshal([]byte(raw.String()), &saved); err != nil {
			return
		}
		webDialect.Lock()
		webDialect.d = webPathDialect{
			prefix:      saved.Prefix,
			slashWins:   saved.SlashWins,
			utf8Wins:    saved.Utf8Wins,
			lowerWins:   saved.LowerWins,
			exactWins:   saved.ExactWins,
			sampleCount: saved.Samples,
		}
		webDialect.Unlock()
		if len(saved.DirCase) > 0 {
			webDirCase.Lock()
			webDirCase.dirs = saved.DirCase
			webDirCase.Unlock()
		}
	})
}

var dialectSaveAt time.Time

func saveDialect() {
	if time.Since(dialectSaveAt) < 5*time.Second {
		return
	}
	dialectSaveAt = time.Now()
	webDialect.Lock()
	d := webDialect.d
	webDialect.Unlock()
	saved := dialectPersist{
		Prefix:    d.prefix,
		SlashWins: d.slashWins,
		Utf8Wins:  d.utf8Wins,
		LowerWins: d.lowerWins,
		ExactWins: d.exactWins,
		Samples:   d.sampleCount,
	}
	webDirCase.Lock()
	saved.DirCase = webDirCase.dirs
	webDirCase.Unlock()
	if saved.Samples == 0 && len(saved.DirCase) == 0 {
		return
	}
	data, err := json.Marshal(saved)
	if err != nil {
		return
	}
	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() {
		return
	}
	storage.Call("setItem", dialectStoreKey, string(data))
}

func learnDirCase(path string) {
	idx := strings.LastIndexAny(path, "/\\")
	if idx <= 0 {
		return
	}
	dir := path[:idx+1]
	webDirCase.Lock()
	if len(webDirCase.dirs) > 512 {
		webDirCase.dirs = make(map[string]string)
	}
	webDirCase.dirs[strings.ToLower(dir)] = dir
	webDirCase.Unlock()
	saveDialect()
}

// rewriteDirCase replaces each known directory prefix with its learned
// casing. The longest matching prefix wins so nested directories work.
func rewriteDirCase(path string) string {
	webDirCase.Lock()
	defer webDirCase.Unlock()
	if len(webDirCase.dirs) == 0 {
		return path
	}
	slashPath := strings.ReplaceAll(path, "\\", "/")
	idx := strings.LastIndex(slashPath, "/")
	if idx <= 0 {
		return path
	}
	dir, file := slashPath[:idx+1], slashPath[idx+1:]
	lower := strings.ToLower(dir)
	if actual, ok := webDirCase.dirs[lower]; ok && actual != dir {
		return actual + file
	}
	return path
}

// candidatePaths returns the URLs to probe for a normalized resource name,
// rooted at the page origin, ordered most-likely-first:
//   - the learned dialect's traits first (after the first successful load),
//   - then UTF-8 forward-slash spellings (what static web hosts serve),
//   - then the raw separator/charset spellings,
//   - case variants last (kRO data references are case-inconsistent; Linux
//     servers 404 the wrong casing, which surfaces as white map tiles).
func (m *Manager) candidatePaths(normalized string) []string {
	loadDialectOnce()
	base := strings.TrimSuffix(m.Root, "/")
	seen := make(map[string]struct{}, 4)
	var candidates []webCandidate
	var add func(path string, slash, utf8, lower, upper bool)
	add = func(path string, slash, utf8, lower, upper bool) {
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		if rewritten := rewriteDirCase(path); rewritten != path {
			// The learned casing of this directory exists on the host —
			// probe it ahead of the raw spellings.
			add(rewritten, slash, utf8, lower, upper)
			return
		}
		c := webCandidate{
			url:   base + "/" + webURLPath(path),
			path:  path,
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
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidateScore(candidates[i]) > candidateScore(candidates[j])
	})
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
