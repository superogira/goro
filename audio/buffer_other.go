//go:build !js || !wasm

package audio

import "time"

// Native builds keep oto's default device buffer: the platform audio
// threads pull from the mux directly and no main-thread tradeoff exists.
const webAudioBufferSize = time.Duration(0)
