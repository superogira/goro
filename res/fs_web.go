//go:build js && wasm

package res

import (
	"fmt"
	"net/url"
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

// candidatePaths returns the URLs to probe for a normalized resource name,
// rooted at the page origin.
func (m *Manager) candidatePaths(normalized string) []string {
	base := strings.TrimSuffix(m.Root, "/")
	seen := make(map[string]struct{}, 3)
	var out []string
	add := func(candidate string) {
		if candidate == "" {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		out = append(out, base+"/"+webURLPath(candidate))
	}
	logical := []string{normalized, strings.ReplaceAll(normalized, "\\", "/")}
	for _, candidate := range withCharsetVariants(logical) {
		add(candidate)
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
