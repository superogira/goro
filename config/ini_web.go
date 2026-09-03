//go:build js && wasm

package config

import (
	"bytes"
	"fmt"
	"syscall/js"
	"time"
)

// applyServerINI fetches goro.ini from the static data directory so the web
// build honors the same settings file as the native one (e.g. the title
// screen's login_bgm_pool). The browser has no local filesystem, so the
// file is downloaded from the page origin. Missing files are fine — the
// defaults and URL parameters still apply.
func applyServerINI(cfg *Config) {
	data := fetchServerINI(defaultDataDirRoot + "/goro.ini")
	if len(data) == 0 {
		return
	}
	if err := applyINI(cfg, bytes.NewReader(data)); err != nil {
		fmt.Printf("server goro.ini: %v\n", err)
	}
}

// fetchServerINI downloads a same-origin URL, bridging the fetch promise
// onto the synchronous caller by polling so the JS event loop keeps running.
func fetchServerINI(path string) []byte {
	type result struct {
		data []byte
		ok   bool
	}
	done := make(chan result, 1)

	onRejected := js.FuncOf(func(_ js.Value, _ []js.Value) any {
		done <- result{}
		return nil
	})
	onFulfilled := js.FuncOf(func(_ js.Value, args []js.Value) any {
		response := args[0]
		if !response.Get("ok").Bool() {
			done <- result{}
			return nil
		}
		bufferPromise := response.Call("arrayBuffer")
		bufferPromise.Call("then", js.FuncOf(func(_ js.Value, args []js.Value) any {
			buffer := args[0]
			size := buffer.Get("byteLength").Int()
			data := make([]byte, size)
			js.CopyBytesToGo(data, js.Global().Get("Uint8Array").New(buffer))
			done <- result{data: data, ok: true}
			return nil
		}), onRejected)
		return nil
	})

	js.Global().Call("fetch", path).Call("then", onFulfilled, onRejected)
	defer onFulfilled.Release()
	defer onRejected.Release()

	deadline := time.Now().Add(10 * time.Second)
	for {
		select {
		case r := <-done:
			if r.ok {
				return r.data
			}
			return nil
		default:
		}
		if time.Now().After(deadline) {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
}
