//go:build js && wasm

package audio

import "time"

// webAudioBufferSize sets the oto device buffer on the WASM build. oto's
// browser driver services AudioWorklet buffer requests on the MAIN thread
// via postMessage round-trips; with the small default buffer that lands
// every few milliseconds and competes with the 60fps game loop, surfacing
// as audio crackle and frame stutter. A ~130ms buffer cuts the main-thread
// audio work by an order of magnitude; the added start latency is
// imperceptible for BGM and acceptable for SFX.
const webAudioBufferSize = 130 * time.Millisecond
