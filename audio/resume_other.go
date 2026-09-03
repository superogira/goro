//go:build !js || !wasm

package audio

import "github.com/ebitengine/oto/v3"

func registerGestureAudioResume(*oto.Context) {}
