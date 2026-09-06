//go:build !(js && wasm)

package glog

import "io"

func consoleLogWriter() io.Writer { return nil }
