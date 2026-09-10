//go:build js && wasm

package ui

import (
	"strconv"
	"strings"
	"syscall/js"

	"github.com/kivutar/goro/client"
)

// serviceWebEnabled reports whether the page provides the DOM service
// window.
func serviceWebEnabled() bool {
	return js.Global().Get("goroServiceSync").Type() == js.TypeFunction
}

// serviceWebSync pushes the list, title, and selection to the page.
func (w *ServiceWindow) serviceWebSync(ctx client.Context) {
	fn := js.Global().Get("goroServiceSync")
	if fn.Type() != js.TypeFunction {
		return
	}
	obj := js.Global().Get("Object").New()
	obj.Set("title", w.title)
	arr := js.Global().Get("Array").New(len(w.services))
	for i, name := range w.services {
		arr.SetIndex(i, name)
	}
	obj.Set("services", arr)
	obj.Set("selected", w.SelectedIndex())
	fn.Invoke(obj)
}

// handleServiceWebAction services one drained "svc:" action.
func (w *ServiceWindow) handleServiceWebAction(ctx client.Context, action string) {
	switch {
	case strings.HasPrefix(action, "svc:sel:"):
		if i, err := strconv.Atoi(strings.TrimPrefix(action, "svc:sel:")); err == nil {
			if i >= 0 && i < len(w.services) && w.selected != nil {
				w.selected.Set(i)
				w.serviceWebSync(ctx)
			}
		}
	case action == "svc:ok":
		w.confirm()
	case action == "svc:cancel":
		w.cancel()
	}
}
