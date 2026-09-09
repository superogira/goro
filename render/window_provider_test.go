package render

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gogpu/gpucontext"
	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
)

type shutdownWindowProvider struct {
	gpucontext.NullWindowProvider
	redraw func()
}

func (p *shutdownWindowProvider) RequestRedraw() { p.redraw() }

type shutdownCloseWindow struct {
	onClose       func() bool
	beforeDestroy func()
	destroyed     atomic.Bool
}

func (w *shutdownCloseWindow) SetOnClose(fn func() bool) { w.onClose = fn }

func (w *shutdownCloseWindow) close() bool {
	if w.onClose != nil && !w.onClose() {
		return false
	}
	if w.beforeDestroy != nil {
		w.beforeDestroy()
	}
	w.destroyed.Store(true)
	return true
}

type shutdownRoot struct {
	*primitives.BoxWidget
	onUnmount func()
}

func (r *shutdownRoot) Mount(widget.Context) {}
func (r *shutdownRoot) Unmount()             { r.onUnmount() }

func TestRunnerCloseDisablesRedrawBeforeUnmount(t *testing.T) {
	var redraws atomic.Int32
	native := &shutdownWindowProvider{
		NullWindowProvider: gpucontext.NullWindowProvider{W: 800, H: 600},
		redraw:             func() { redraws.Add(1) },
	}
	provider := &uiWindowProvider{WindowProvider: native}
	ui := uiapp.New(uiapp.WithWindowProvider(provider))
	r := &runner{ui: ui, uiWindow: provider}
	t.Cleanup(r.close)
	unmounts := 0
	ui.SetRoot(&shutdownRoot{
		BoxWidget: primitives.Box(),
		onUnmount: func() {
			unmounts++
			// Widget cleanup may invalidate the UI as well as animation ticks.
			provider.RequestRedraw()
		},
	})
	provider.RequestRedraw()
	beforeClose := redraws.Load()
	if beforeClose == 0 {
		t.Fatal("redraw was not forwarded while the window was open")
	}

	r.close()
	r.close() // OnClose and Run's deferred cleanup share the same path.
	if unmounts != 1 || ui.Window().Root() != nil {
		t.Fatalf("UI was not closed exactly once: unmounts=%d, root=%v", unmounts, ui.Window().Root())
	}

	// A pumper that is still finishing a tick must never wake the destroyed
	// native window, even if it sends multiple requests from other goroutines.
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			for range 100 {
				provider.RequestRedraw()
			}
		})
	}
	workers.Wait()
	if got := redraws.Load(); got != beforeClose {
		t.Fatalf("shutdown forwarded %d redraws to the native window", got-beforeClose)
	}
}

func TestUIWindowProviderNativeCloseStopsRedrawBeforeDestroy(t *testing.T) {
	var redraws atomic.Int32
	provider := &uiWindowProvider{WindowProvider: &shutdownWindowProvider{
		redraw: func() { redraws.Add(1) },
	}}
	ui := uiapp.New(uiapp.WithWindowProvider(provider))
	ui.SetRoot(primitives.Box())
	r := &runner{ui: ui, uiWindow: provider}
	t.Cleanup(r.close)
	provider.RequestRedraw()
	beforeClose := redraws.Load()
	window := &shutdownCloseWindow{beforeDestroy: func() {
		// GoGPU destroys the native window after accepting the close request,
		// then invokes App.OnClose later. Simulate an animation tick in that gap.
		provider.RequestRedraw()
		if got := redraws.Load(); got != beforeClose {
			t.Errorf("native teardown began while UI redraws were still enabled")
		}
		if ui.Window().Root() == nil {
			t.Error("pre-close callback performed UI cleanup before App.OnClose")
		}
	}}
	provider.guardClose(window)
	if window.onClose == nil {
		t.Fatal("native pre-close callback was not installed")
	}
	if !window.close() || !window.destroyed.Load() {
		t.Fatal("native close request was not accepted")
	}
	ui.Window().Context().Invalidate()
	if got := redraws.Load(); got != beforeClose {
		t.Error("UI invalidation reached the destroyed native window")
	}
	r.close()
	if ui.Window().Root() != nil {
		t.Error("App.OnClose did not finish UI cleanup")
	}
}

func TestUIWindowProviderNativeCloseWaitsForActiveRedraw(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	provider := &uiWindowProvider{WindowProvider: &shutdownWindowProvider{
		redraw: func() {
			close(entered)
			<-release
		},
	}}
	window := &shutdownCloseWindow{}
	provider.guardClose(window)
	go provider.RequestRedraw()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("redraw did not enter the native window")
	}
	closing := make(chan struct{})
	closed := make(chan struct{})
	go func() {
		close(closing)
		if !window.close() {
			t.Error("native close request was rejected")
		}
		close(closed)
	}()
	<-closing
	select {
	case <-closed:
		t.Fatal("close returned while a native redraw was still in flight")
	case <-time.After(20 * time.Millisecond):
	}
	unblock()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("close did not finish after the redraw returned")
	}
	if !window.destroyed.Load() {
		t.Fatal("native window was not destroyed after the redraw returned")
	}
	// Forwarding this would re-enter the native callback and panic.
	provider.RequestRedraw()
}

func TestUIWindowProviderTracksWindowGeometryChanges(t *testing.T) {
	native := &gpucontext.NullWindowProvider{W: 800, H: 600, SF: 1}
	provider := &uiWindowProvider{WindowProvider: native}
	for _, size := range []gpucontext.NullWindowProvider{
		{W: 800, H: 600, SF: 1},
		{W: 1920, H: 1080, SF: 1.5},
		{W: 800, H: 600, SF: 1},
	} {
		*native = size
		width, height := provider.Size()
		if width != size.W || height != size.H || provider.ScaleFactor() != size.SF {
			t.Fatalf("window geometry did not track fullscreen/DPI change: %dx%d @ %g", width, height, provider.ScaleFactor())
		}
	}
}
