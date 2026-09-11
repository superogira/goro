//go:build !js || !wasm

package game

import "github.com/kivutar/goro/render"

// cursorWebEnabled is false on native: the cursor draws on canvas.
func cursorWebEnabled() bool { return false }

func cursorWebSync(img *render.Image, key string, x, y float64) {}

func cursorWebHide() {}
