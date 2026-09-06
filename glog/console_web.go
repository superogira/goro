//go:build js && wasm

package glog

import (
	"io"
	"strings"
	"syscall/js"
)

// consoleLogWriter routes log output to the browser console on wasm, where
// os.Stderr is not connected to anything. A page-defined window.__goroLog
// function receives every line first (test harness hook); when absent the
// line goes to console.log.
func consoleLogWriter() io.Writer { return consoleWriter{} }

type consoleWriter struct{}

func (consoleWriter) Write(p []byte) (int, error) {
	s := strings.TrimRight(string(p), "\r\n")
	if s == "" {
		return len(p), nil
	}
	g := js.Global()
	if fn := g.Get("__goroLog"); fn.Type() == js.TypeFunction {
		fn.Invoke(s)
	} else {
		g.Get("console").Call("log", s)
	}
	return len(p), nil
}
