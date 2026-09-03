//go:build js && wasm

package audio

// waitForAudioReady is a no-op on the web: oto's ready channel stays open
// until the browser resumes the AudioContext after a user gesture, and
// blocking the game on it froze boot behind a "touch the screen once" gate.
func waitForAudioReady(<-chan struct{}) {}
