//go:build js && wasm

package config

// defaultAsyncUI keeps UI rasterization on the draw thread in the browser:
// the async rasterizer worker parks in the single-threaded wasm runtime and
// dirty regions never flush, leaving the UI blank.
func defaultAsyncUI() bool { return false }
