package social

import gameui "github.com/kivutar/goro/ui"

func newTestSystem() *System {
	s := &System{}
	s.Bind(Bindings{Windows: &Windows{}, Console: &gameui.ChatConsole{}, Minimap: &gameui.Minimap{}})
	return s
}
