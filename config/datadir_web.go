//go:build js && wasm

package config

// defaultDataDirRoot is the browser default: the static server exposes the
// extracted Ragnarok data folder at this absolute path on the page origin.
const defaultDataDirRoot = "/webro"
