package render

import (
	"sync"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/gpucontext"
)

// uiWindowProvider prevents background UI redraws from reaching a native
// window after teardown. Geometry still comes from the live window so
// fullscreen and DPI changes propagate normally.
type uiWindowProvider struct {
	gpucontext.WindowProvider
	mu     sync.RWMutex
	closed bool
}

func (p *uiWindowProvider) RequestRedraw() {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if !p.closed {
		p.WindowProvider.RequestRedraw()
	}
}

// guardClose drains redraws before a native close request is accepted.
// App.OnClose runs after window destruction on the X/Alt+F4 path.
func (p *uiWindowProvider) guardClose(window gogpu.PlatformWindowCloser) {
	window.SetOnClose(func() bool {
		p.close()
		return true
	})
}

// close waits for in-flight redraws and rejects all subsequent requests.
func (p *uiWindowProvider) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
}
