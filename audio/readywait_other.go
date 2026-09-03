//go:build !js || !wasm

package audio

// waitForAudioReady blocks until the native audio driver is initialized.
// Handing the context out early silences every player created from it.
func waitForAudioReady(ready <-chan struct{}) {
	<-ready
}
