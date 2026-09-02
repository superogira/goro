//go:build !js || !wasm

package config

// defaultDataDirRoot is the empty default on native builds: resolveDataDir
// falls back to the working directory when no data dir is configured.
const defaultDataDirRoot = ""
