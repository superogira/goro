//go:build linux

package platform

// fbdev backend: fullscreen rendering straight into /dev/fb0 with evdev
// input, for embedded Linux handhelds (Anbernic RG35XX & co.) that run
// without an X11/Wayland compositor. Selected at runtime with
// GOGPU_PLATFORM=fbdev.
//
// The window is the whole framebuffer. Frames arrive through BlitPixels
// (called by the renderer after presenting to the headless software
// surface) and are swizzled into the fb layout from the var screeninfo.
// Input is read from /dev/input/event*; gamepad sticks drive a virtual
// mouse cursor and buttons map to mouse buttons and common keys:
//
//	A/B/X       -> left/right/middle click
//	Y           -> Space        Start -> Enter      Select -> Escape
//	L1/R1/L2/R2 -> F1/F2/F3/F4  L3 -> Tab           R3 -> Backspace
//	D-pad       -> arrow keys   both sticks         -> cursor movement
//
// Development overrides (no real fb/ioctl hardware needed):
//
//	GOGPU_FB=/path/to/file  use a plain file instead of /dev/fb0
//	GOGPU_FB_WIDTH / _HEIGHT / _BPP   geometry for that file
//	GOGPU_INPUT_GLOB        evdev glob (default /dev/input/event*)
//	GOGPU_INPUT_NONE=1      do not open any input device

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/gogpu/gogpu/internal/platform/eventqueue"
	"github.com/gogpu/gpucontext"
	"golang.org/x/sys/unix"
)

const (
	fbIOCTLGetVarScreeninfo = 0x4600
	fbIOCTLGetFixScreeninfo = 0x4602
	fbIOCTLPanDisplay       = 0x4606

	evTypeKey = 0x01
	evTypeAbs = 0x03

	evAbsX     = 0x00
	evAbsY     = 0x01
	evAbsRX    = 0x03
	evAbsRY    = 0x04
	evAbsHat0X = 0x10
	evAbsHat0Y = 0x11

	keyEscape     = 1
	key1          = 2
	key0          = 11
	keyMinus      = 12
	keyEqual      = 13
	keyBackspace  = 14
	keyTab        = 15
	keyQ          = 16
	keyW          = 17
	keyE          = 18
	keyR          = 19
	keyT          = 20
	keyY          = 21
	keyU          = 22
	keyI          = 23
	keyO          = 24
	keyP          = 25
	keyLeftBrace  = 26
	keyRightBrace = 27
	keyEnter      = 28
	keyLeftCtrl   = 29
	keyA          = 30
	keyS          = 31
	keyD          = 32
	keyF          = 33
	keyG          = 34
	keyH          = 35
	keyJ          = 36
	keyK          = 37
	keyL          = 38
	keySemicolon  = 39
	keyApostrophe = 40
	keyGrave      = 41
	keyLeftShift  = 42
	keyBackslash  = 43
	keyZ          = 44
	keyX          = 45
	keyC          = 46
	keyV          = 47
	keyB          = 48
	keyN          = 49
	keyM          = 50
	keyComma      = 51
	keyDot        = 52
	keySlash      = 53
	keyRightShift = 54
	keyLeftAlt    = 56
	keySpace      = 57
	keyCapsLock   = 58
	keyF1         = 59
	keyF10        = 68
	keyRightCtrl  = 97
	keyHome       = 102
	keyUp         = 103
	keyPageUp     = 104
	keyLeft       = 105
	keyRight      = 106
	keyEnd        = 107
	keyDown       = 108
	keyPageDown   = 109
	keyInsert     = 110
	keyDelete     = 111
	keyF11        = 87
	keyF12        = 88
	keyRightAlt   = 100

	btnLeft   = 0x110
	btnRight  = 0x111
	btnMiddle = 0x112

	// Physical layout decoded from the user's presses on this ANBERNIC-keys
	// device: face buttons occupy 0x130-0x133, then L1=0x134, R1=0x135,
	// L2=0x136, R2=0x137, and 0x138 — the ninth key — is MENU (held 3.5s
	// twice while testing). The standard BTN_TL/BTN_MODE numbering does
	// NOT match this hardware.
	btnSouth  = 0x130 // A
	btnEast   = 0x131 // B
	btnNorth  = 0x133 // X
	btnL1     = 0x134 // physical L1 (BTN_WEST slot on standard pads)
	btnR1     = 0x135 // physical R1 (BTN_TL slot on standard pads)
	btnL2     = 0x136 // physical L2 (BTN_TR slot on standard pads)
	btnR2     = 0x137 // physical R2 (BTN_TL2 slot on standard pads)
	btnMenu   = 0x138 // physical MENU (BTN_TR2 slot on standard pads)
	btnSelect = 0x139
	btnStart  = 0x13a
	btnMode   = 0x13d
	btnThumbl = 0x13b
	btnThumbr = 0x13c

	// Side keys seen in the field: VOLUMEDOWN/VOLUMEUP on the gamepad device
	// (event1), and the PMIC power key on event0 (axp2202-pek). The power
	// key toggles screen+audio off without sleeping — the game keeps its
	// connection.
	keyVolDown = 0x72 // KEY_VOLUMEDOWN
	keyVolUp   = 0x73 // KEY_VOLUMEUP
	keyPower   = 0x74 // KEY_POWER

	// btnSel0 is SELECT on this device — the one non-gamepad code the
	// ANBERNIC-keys device advertises (from its /proc caps bitmap).
	btnSel0 = 0x162
	keyEsc  = 0x001

	// Stick handling: RG35XX-style pads report 0..255 with center ~128.
	stickCenter = 127.5
	stickRange  = 127.5
	cursorSpeed = 650.0 // px/s at full deflection
	cursorTick  = time.Second / 120
)

// fbGeometry is the subset of fb_var/fb_fix screeninfo the blitter needs.
type fbGeometry struct {
	width, height    int
	bpp              int
	lineLength       int
	xoffset, yoffset int
	widthVirtual     int
	heightVirtual    int
	redBits          fbBitfield
	greenBits        fbBitfield
	blueBits         fbBitfield
	smemLen          int
}

type fbBitfield struct {
	offset, length uint32
}

// fbdevPlatform is the process-level backend: one fullscreen window,
// evdev readers feeding a shared event queue.
type fbdevPlatform struct {
	events *eventqueue.Queue[Event]
	wakeCh chan struct{}
	geo    fbGeometry
	fbFile *os.File
	fbMem  []byte

	mu      sync.Mutex // guards window creation/close state
	window  *fbdevWindow
	started time.Time

	inputMu sync.Mutex
	inputs  []*os.File
	closing bool

	// virtual cursor / button / stick state, guarded by inputMu
	cursorX, cursorY float64
	// fbScale renders at 1/fbScale of the panel and BlitPixels upscales;
	// pointer events are mapped back into the smaller surface space.
	fbScale int
	// varBuf is the raw fb_var_screeninfo from probing; panning with it
	// at init activates the fb layer for scanout (StockOS leaves the
	// layer off until an FBIOPAN/FBIOPUT touches the mode).
	varBuf [160]byte
	// gles enables GPU rendering through EGL on the framebuffer (the
	// sunxi Mali stack); eglWin backs eglCreateWindowSurface.
	gles     bool
	eglWin   fbdevEGLWindow
	eglWinOK bool
	// held maps currently-pressed evdev button codes to press time; the
	// quit watcher closes the window once a quit combo stays held long
	// enough (MENU / SELECT / SELECT+START / L1+R1 — several paths so a
	// mis-mapped MENU button can never trap the user inside the app).
	held             map[uint16]time.Time
	seenCodes        map[uint16]bool
	keyTraces        map[string]int
	// The firmware emits a SELECT (0x162) press right after every MENU
	// (0x138) release. Swallow that echo so a MENU tap does not also open
	// the in-game escape menu.
	suppressSel0Until time.Time
	// Power-key panel cycle: on → dim (backlight off, mute) → off (fb
	// blanked too) → on (restore). Not a sleep — sockets stay alive.
	powerState          int
	dispdbgRestoreLevel int
	buttons             gpucontext.Buttons
	axes                map[uint16]int32
	hatX, hatY          int
	shift, ctrl, alt  bool
}

func newFBDevPlatform() PlatformManager {
	return &fbdevPlatform{
		events:  eventqueue.New[Event](eventqueue.DefaultCapacity),
		wakeCh:  make(chan struct{}, 1),
		axes:    make(map[uint16]int32),
		started: time.Now(),
	}
}

// --- PlatformManager ---

func (p *fbdevPlatform) Init() error {
	dev := os.Getenv("GOGPU_FB")
	if dev == "" {
		dev = "/dev/fb0"
	}
	f, err := os.OpenFile(dev, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("fbdev: open %s: %w", dev, err)
	}
	geo, err := p.probeFBGeometry(f)
	if err != nil {
		f.Close()
		return fmt.Errorf("fbdev: %w", err)
	}
	size := geo.smemLen
	if size == 0 {
		size = geo.lineLength * (geo.yoffset + geo.height)
	}
	if size == 0 {
		f.Close()
		return fmt.Errorf("fbdev: framebuffer has zero size")
	}
	mem, err := unix.Mmap(int(f.Fd()), 0, size, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		f.Close()
		return fmt.Errorf("fbdev: mmap: %w", err)
	}
	p.fbFile, p.fbMem, p.geo = f, mem, geo
	p.fbScale = fbScaleFromEnv()
	p.gles = os.Getenv("GOGPU_FB_GLES") == "1"
	if p.gles {
		w, h := p.surfaceSize()
		p.eglWin = fbdevEGLWindow{Width: uint16(w), Height: uint16(h)}
		p.eglWinOK = true
	}
	logger().Info("fbdev: framebuffer ready",
		"device", dev, "size", fmt.Sprintf("%dx%d", geo.width, geo.height),
		"virtual", fmt.Sprintf("%dx%d", geo.widthVirtual, geo.heightVirtual),
		"offsets", fmt.Sprintf("+%d+%d", geo.xoffset, geo.yoffset),
		"bpp", geo.bpp, "line", geo.lineLength, "maplen", len(mem),
		"bitfields", fmt.Sprintf("r<%d:%d> g<%d:%d> b<%d:%d>",
			geo.redBits.offset, geo.redBits.length,
			geo.greenBits.offset, geo.greenBits.length,
			geo.blueBits.offset, geo.blueBits.length))
	// Activate the fb layer for scanout: on StockOS the layer stays off
	// until an FBIOPAN/FBIOPUT touches the mode — writes before this are
	// invisible (fbtest proved the pan is the trigger).
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, f.Fd(), fbIOCTLPanDisplay, uintptr(unsafe.Pointer(&p.varBuf[0]))); errno != 0 {
		logger().Info("fbdev: FBIOPAN_DISPLAY failed (continuing)", "errno", errno.Error())
	} else {
		logger().Info("fbdev: FBIOPAN_DISPLAY ok, fb layer activated")
	}
	if os.Getenv("GOGPU_FB_PROBE") == "color" {
		p.probeBufferColors()
	}
	p.held = make(map[uint16]time.Time)
	p.seenCodes = make(map[uint16]bool)
	p.startInput()
	go p.cursorLoop()
	go p.quitWatcher()
	return nil
}

func (p *fbdevPlatform) CreateWindow(config Config) (PlatformWindow, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.window != nil {
		return nil, fmt.Errorf("fbdev: only one window is supported")
	}
	w := &fbdevWindow{
		platform: p,
		id:       NewWindowID(),
	}
	p.window = w
	p.inputMu.Lock()
	p.cursorX = float64(p.geo.width) / 2
	p.cursorY = float64(p.geo.height) / 2
	p.inputMu.Unlock()
	// Size the application to the (optionally scaled) surface and grant
	// focus so text fields work without a window manager.
	surfW, surfH := p.surfaceSize()
	p.events.Push(Event{Type: EventResize, WindowID: w.id,
		Width: surfW, Height: surfH,
		PhysicalWidth: surfW, PhysicalHeight: surfH})
	p.events.Push(Event{Type: EventFocus, WindowID: w.id, Focused: true})
	p.wake()
	return w, nil
}

func (p *fbdevPlatform) PollEvents() Event {
	ev, ok := p.events.Pop()
	if !ok {
		return Event{Type: EventNone}
	}
	return ev
}

func (p *fbdevPlatform) WaitEvents() {
	if p.events.Len() > 0 {
		return
	}
	<-p.wakeCh
}

func (p *fbdevPlatform) WakeUp() { p.wake() }

func (p *fbdevPlatform) wake() {
	select {
	case p.wakeCh <- struct{}{}:
	default:
	}
}

// push posts an event from any goroutine.
func (p *fbdevPlatform) push(ev Event) {
	p.events.Push(ev)
	p.wake()
}

func (p *fbdevPlatform) ClipboardRead() (string, error) { return "", nil }
func (p *fbdevPlatform) ClipboardWrite(string) error    { return nil }
func (p *fbdevPlatform) DarkMode() bool                 { return false }
func (p *fbdevPlatform) ReduceMotion() bool             { return false }
func (p *fbdevPlatform) HighContrast() bool             { return false }
func (p *fbdevPlatform) FontScale() float32             { return 1 }
func (p *fbdevPlatform) SubpixelLayout() gpucontext.SubpixelLayout {
	return gpucontext.SubpixelNone
}
func (p *fbdevPlatform) FontSmoothing() gpucontext.FontSmoothing {
	return gpucontext.FontSmoothingGrayscale
}
func (p *fbdevPlatform) SetAppName(string) {}

func (p *fbdevPlatform) ShowOpenFileDialog(FileDialogOptions) ([]string, error) {
	return nil, nil
}

func (p *fbdevPlatform) ShowSaveFileDialog(FileDialogOptions) (string, error) {
	return "", nil
}

func (p *fbdevPlatform) Destroy() {
	p.inputMu.Lock()
	p.closing = true
	for _, f := range p.inputs {
		f.Close()
	}
	p.inputs = nil
	p.inputMu.Unlock()
	if p.fbMem != nil {
		unix.Munmap(p.fbMem)
		p.fbMem = nil
	}
	if p.fbFile != nil {
		p.fbFile.Close()
		p.fbFile = nil
	}
}

// --- framebuffer geometry ---

func (p *fbdevPlatform) probeFBGeometry(f *os.File) (fbGeometry, error) {
	var geo fbGeometry
	// Development fake: a plain file standing in for the framebuffer,
	// with geometry supplied through env vars instead of ioctls.
	if w := os.Getenv("GOGPU_FB_WIDTH"); w != "" {
		if _, err := fmt.Sscanf(w, "%d", &geo.width); err != nil {
			return geo, err
		}
		if h := os.Getenv("GOGPU_FB_HEIGHT"); h != "" {
			if _, err := fmt.Sscanf(h, "%d", &geo.height); err != nil {
				return geo, err
			}
		}
		geo.bpp = 32
		if b := os.Getenv("GOGPU_FB_BPP"); b != "" {
			if _, err := fmt.Sscanf(b, "%d", &geo.bpp); err != nil {
				return geo, err
			}
		}
		geo.lineLength = geo.width * geo.bpp / 8
		geo.redBits = fbBitfield{offset: 16, length: 8}
		geo.greenBits = fbBitfield{offset: 8, length: 8}
		geo.blueBits = fbBitfield{offset: 0, length: 8}
		geo.smemLen = geo.lineLength * geo.height
		if geo.width <= 0 || geo.height <= 0 {
			return geo, fmt.Errorf("bad fake fb geometry %dx%d", geo.width, geo.height)
		}
		return geo, nil
	}

	varBuf := &p.varBuf
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, f.Fd(), fbIOCTLGetVarScreeninfo, uintptr(unsafe.Pointer(&varBuf[0]))); errno != 0 {
		return geo, fmt.Errorf("FBIOGET_VSCREENINFO: %w", errno)
	}
	geo.width = int(binary.LittleEndian.Uint32(varBuf[0:]))
	geo.height = int(binary.LittleEndian.Uint32(varBuf[4:]))
	geo.widthVirtual = int(binary.LittleEndian.Uint32(varBuf[8:]))
	geo.heightVirtual = int(binary.LittleEndian.Uint32(varBuf[12:]))
	geo.xoffset = int(binary.LittleEndian.Uint32(varBuf[16:]))
	geo.yoffset = int(binary.LittleEndian.Uint32(varBuf[20:]))
	geo.bpp = int(binary.LittleEndian.Uint32(varBuf[24:]))
	geo.redBits = fbBitfield{binary.LittleEndian.Uint32(varBuf[32:]), binary.LittleEndian.Uint32(varBuf[36:])}
	geo.greenBits = fbBitfield{binary.LittleEndian.Uint32(varBuf[44:]), binary.LittleEndian.Uint32(varBuf[48:])}
	geo.blueBits = fbBitfield{binary.LittleEndian.Uint32(varBuf[56:]), binary.LittleEndian.Uint32(varBuf[60:])}

	var fixBuf [80]byte
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, f.Fd(), fbIOCTLGetFixScreeninfo, uintptr(unsafe.Pointer(&fixBuf[0]))); errno != 0 {
		// Some fbdev drivers refuse the fixed-screeninfo ioctl; the
		// variable info plus the classic packed layout is enough.
		logger().Warn("fbdev: FBIOGET_FSCREENINFO failed, assuming packed layout", "errno", errno.Error())
	} else {
		geo.smemLen = int(binary.LittleEndian.Uint32(fixBuf[24:]))
		geo.lineLength = int(binary.LittleEndian.Uint32(fixBuf[48:]))
	}
	if geo.lineLength == 0 {
		geo.lineLength = geo.width * geo.bpp / 8
	}
	if geo.width == 0 || geo.height == 0 || geo.bpp == 0 {
		return geo, fmt.Errorf("unexpected fb geometry %dx%d bpp %d", geo.width, geo.height, geo.bpp)
	}
	return geo, nil
}

// --- window ---

type fbdevWindow struct {
	platform    *fbdevPlatform
	id          WindowID
	shouldClose bool
	cursorMode  int
}

func (w *fbdevWindow) ID() WindowID { return w.id }

// GetHandle returns zero handles: the renderer creates a headless
// software surface instead of a platform GPU surface.
func (w *fbdevWindow) GetHandle() (instance, window uintptr) { return 0, 0 }

// UseHeadlessSurface marks the window for the renderer: present goes
// through the software surface and BlitPixels.
func (w *fbdevWindow) UseHeadlessSurface() bool { return !w.platform.gles }

// FbdevEGLWindow returns the native fbdev_window for EGL when the GLES
// path is enabled; ok is false in software mode.
func (w *fbdevWindow) FbdevEGLWindow() (window uintptr, ok bool) {
	if !w.platform.gles || !w.platform.eglWinOK {
		return 0, false
	}
	return uintptr(unsafe.Pointer(&w.platform.eglWin)), true
}

// fbdevEGLWindow mirrors the sunxi Mali EGL native window: the driver
// reads the width/height at surface creation.
type fbdevEGLWindow struct {
	Width, Height uint16
}

func (w *fbdevWindow) LogicalSize() (int, int) {
	w2, h2 := w.platform.surfaceSize()
	return w2, h2
}
func (w *fbdevWindow) PhysicalSize() (int, int) {
	w2, h2 := w.platform.surfaceSize()
	return w2, h2
}
func (w *fbdevWindow) ScaleFactor() float64 { return 1 }

func (w *fbdevWindow) PrepareFrame() PrepareFrameResult {
	// Report the (optionally scaled) surface size, not the raw panel
	// size: beginFrame reconfigures the surface to match this value, so
	// returning the panel size here fought the scaled resize every
	// frame (surface ping-ponged 320x240 <-> 640x480).
	width, height := w.platform.surfaceSize()
	return PrepareFrameResult{
		ScaleFactor:    1,
		PhysicalWidth:  uint32(width),  //nolint:gosec // fb size fits u32
		PhysicalHeight: uint32(height), //nolint:gosec // fb size fits u32
	}
}

func (w *fbdevWindow) InSizeMove() bool    { return false }
func (w *fbdevWindow) ShouldClose() bool   { return w.shouldClose }
func (w *fbdevWindow) SetTitle(string)     {}
func (w *fbdevWindow) SetMinSize(int, int) {}
func (w *fbdevWindow) SetMaxSize(int, int) {}
func (w *fbdevWindow) SetCursor(int)       {}
func (w *fbdevWindow) SetFrameless(bool)   {}
func (w *fbdevWindow) IsFrameless() bool   { return true }
func (w *fbdevWindow) SetFullscreen(bool)  {}
func (w *fbdevWindow) IsFullscreen() bool  { return true }

func (w *fbdevWindow) SetHitTestCallback(func(x, y float64) gpucontext.HitTestResult) {}
func (w *fbdevWindow) Minimize()                                                      {}
func (w *fbdevWindow) Maximize()                                                      {}
func (w *fbdevWindow) IsMaximized() bool                                              { return false }

func (w *fbdevWindow) Close() {
	w.shouldClose = true
	w.platform.push(Event{Type: EventClose, WindowID: w.id})
}

func (w *fbdevWindow) Show() {}
func (w *fbdevWindow) Hide() {}

func (w *fbdevWindow) SetPosition(int, int)                 {}
func (w *fbdevWindow) SyncFrame()                           {}
func (w *fbdevWindow) SetCursorMode(mode int)               { w.cursorMode = mode }
func (w *fbdevWindow) CursorMode() int                      { return w.cursorMode }
func (w *fbdevWindow) SetModalFrameCallback(func())         {}
func (w *fbdevWindow) RequestSize(width, height int)        {}
func (w *fbdevWindow) StartDrag([]string, func(DragResult)) {}
func (w *fbdevWindow) Destroy()                             {}

// probeBufferColors paints each screen-sized chunk of the framebuffer
// mapping a distinct solid color (red, green, blue, yellow) and holds for
// a few seconds: whichever color the panel shows identifies the buffer
// the display actually scans — these firmwares expose a multi-buffered
// fb (yres_virtual > yres, panning via yoffset).
func (p *fbdevPlatform) probeBufferColors() {
	geo := p.geo
	screen := geo.lineLength * geo.height
	if screen <= 0 {
		return
	}
	// Cycle the palette over EVERY screen-sized chunk of the mapping —
	// if the panel scans any 480-line window inside smem, a color shows.
	colors := [][3]byte{{255, 0, 0}, {0, 255, 0}, {0, 0, 255}, {255, 255, 0}}
	chunks := len(p.fbMem) / screen
	for i := 0; i < chunks; i++ {
		seg := p.fbMem[i*screen:]
		if len(seg) > screen {
			seg = seg[:screen]
		}
		c := colors[i%len(colors)]
		fillFBColor(seg, geo, c[0], c[1], c[2])
	}
	logger().Info("fbdev: color probe painted",
		"chunks", chunks, "screen_bytes", screen,
		"hold", "6s", "order", "red,green,blue,yellow,cycling")
	time.Sleep(6 * time.Second)
}

// fillFBColor fills a framebuffer region with one color, honoring the
// bitfield layout the driver reported.
func fillFBColor(dst []byte, geo fbGeometry, r, g, b byte) {
	switch geo.bpp {
	case 32:
		var pix uint32
		if geo.redBits.length > 0 {
			pix = (uint32(r) >> (8 - geo.redBits.length)) << geo.redBits.offset
			pix |= (uint32(g) >> (8 - geo.greenBits.length)) << geo.greenBits.offset
			pix |= (uint32(b) >> (8 - geo.blueBits.length)) << geo.blueBits.offset
		} else {
			pix = uint32(r)<<16 | uint32(g)<<8 | uint32(b)
		}
		for off := 0; off+4 <= len(dst); off += 4 {
			binary.LittleEndian.PutUint32(dst[off:], pix)
		}
	case 16:
		pix := uint16((uint32(r)>>3)<<11 | (uint32(g)>>2)<<5 | uint32(b)>>3)
		for off := 0; off+2 <= len(dst); off += 2 {
			binary.LittleEndian.PutUint16(dst[off:], pix)
		}
	}
}

// BlitPixels writes a presented software frame (RGBA or BGRA, row-major,
// tightly packed) into the framebuffer, honoring offsets and line length.
// A frame that is a whole fraction of the panel is scaled up with
// nearest-neighbor sampling, so a reduced render resolution still fills
// the screen.
func (w *fbdevWindow) BlitPixels(pixels []byte, width, height int, bgra bool) error {
	p := w.platform
	if p.gles {
		// EGL presents straight to the framebuffer; no CPU blit.
		return nil
	}
	geo := p.geo
	if p.fbMem == nil {
		return fmt.Errorf("fbdev: framebuffer not mapped")
	}
	if width <= 0 || height <= 0 {
		return fmt.Errorf("fbdev: blit size %dx%d", width, height)
	}
	stepX, stepY := 1, 1
	if geo.width > width && geo.width%width == 0 && geo.height%height == 0 && geo.height >= height {
		stepX = geo.width / width
		stepY = geo.height / height
	}
	srcRow := width * 4
	dstBase := geo.yoffset*geo.lineLength + geo.xoffset*geo.bpp/8
	// Iterate DESTINATION rows: with stepY > 1 the old source-row loop
	// only wrote every stepY-th fb row, leaving the skipped rows showing
	// whatever was there before (fbtest's red, on the device).
	for dy := 0; dy < geo.height; dy++ {
		sy := dy / stepY
		if sy >= height {
			break
		}
		src := pixels[sy*srcRow : sy*srcRow+srcRow]
		row := p.fbMem[dstBase+dy*geo.lineLength:]
		switch geo.bpp {
		case 32:
			if stepX == 1 {
				for x := 0; x < width && x < geo.width; x++ {
					r, g, b := src[x*4], src[x*4+1], src[x*4+2]
					if bgra {
						r, b = b, r
					}
					// Standard little-endian XRGB (bytes B,G,R,X): fbtest
					// proved the panel takes this literally; the driver's
					// declared bitfields disagree and swap red/blue.
					pix := uint32(r)<<16 | uint32(g)<<8 | uint32(b)
					binary.LittleEndian.PutUint32(row[x*4:], pix)
				}
				break
			}
			for x := 0; x < geo.width; x++ {
				sx := x / stepX
				if sx >= width {
					break
				}
				r, g, b := src[sx*4], src[sx*4+1], src[sx*4+2]
				if bgra {
					r, b = b, r
				}
				var pix uint32 = uint32(r)<<16 | uint32(g)<<8 | uint32(b)
				binary.LittleEndian.PutUint32(row[x*4:], pix)
			}
		case 16:
			if stepX == 1 {
				for x := 0; x < width && x < geo.width; x++ {
					r, g, b := src[x*4], src[x*4+1], src[x*4+2]
					if bgra {
						r, b = b, r
					}
					pix := (uint32(r)>>3)<<11 | (uint32(g)>>2)<<5 | uint32(b)>>3
					binary.LittleEndian.PutUint16(row[x*2:], uint16(pix))
				}
				break
			}
			for x := 0; x < geo.width; x++ {
				sx := x / stepX
				if sx >= width {
					break
				}
				r, g, b := src[sx*4], src[sx*4+1], src[sx*4+2]
				if bgra {
					r, b = b, r
				}
				pix := (uint32(r)>>3)<<11 | (uint32(g)>>2)<<5 | uint32(b)>>3
				binary.LittleEndian.PutUint16(row[x*2:], uint16(pix))
			}
		default:
			return fmt.Errorf("fbdev: unsupported bpp %d", geo.bpp)
		}
	}
	return nil
}

// --- input ---

func (p *fbdevPlatform) startInput() {
	if os.Getenv("GOGPU_INPUT_NONE") == "1" {
		return
	}
	// The kernel's own input inventory: one block per evdev device with its
	// name, handlers (eventN) and KEY/BTN capability bitmask. Decoding the
	// mask tells us exactly which button codes the hardware can send — the
	// ground truth when hunting a button that never reaches handleKey.
	if b, err := os.ReadFile("/proc/bus/input/devices"); err == nil {
		logger().Info("fbdev: kernel input inventory (/proc/bus/input/devices)", "dump", string(b))
	} else {
		logger().Info("fbdev: /proc/bus/input/devices unavailable", "err", err.Error())
	}
	glob := os.Getenv("GOGPU_INPUT_GLOB")
	if glob == "" {
		glob = "/dev/input/event*"
	}
	devs, _ := filepath.Glob(glob)
	opened := 0
	for _, d := range devs {
		f, err := os.OpenFile(d, os.O_RDONLY|unix.O_NONBLOCK, 0)
		if err != nil {
			logger().Info("fbdev: input device open failed", "device", d, "err", err.Error())
			continue
		}
		p.probeEvdevGrab(f, d)
		p.inputMu.Lock()
		p.inputs = append(p.inputs, f)
		p.inputMu.Unlock()
		go p.readEvents(f)
		opened++
		logger().Info("fbdev: input device listening", "device", d)
	}
	if opened > 0 {
		logger().Info("fbdev: listening to input devices", "count", opened)
	}
}

// probeEvdevGrab tests whether another process holds an exclusive
// EVIOCGRAB on the device: grabbing ourselves succeeds only when nobody
// else has. An EBUSY answer names the reason no events ever reach us.
// The grab is released immediately either way.
func (p *fbdevPlatform) probeEvdevGrab(f *os.File, device string) {
	const eviocGrab = 0x4590 // _IO('E', 0x90), dir=none size=0
	grab := int32(1)
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, f.Fd(), eviocGrab, uintptr(unsafe.Pointer(&grab))); errno != 0 {
		logger().Info("fbdev: grab probe — device already grabbed by another process",
			"device", device, "errno", errno.Error())
		return
	}
	release := int32(0)
	unix.Syscall(unix.SYS_IOCTL, f.Fd(), eviocGrab, uintptr(unsafe.Pointer(&release)))
	logger().Info("fbdev: grab probe — device not grabbed (events should broadcast)", "device", device)
}

// readEvents parses one evdev device. struct input_event on 64-bit Linux
// is {i64 sec, i64 usec, u16 type, u16 code, i32 value} = 24 bytes — the
// timeval alone is 16. evdev rejects reads smaller than one event with
// EINVAL, so a 16-byte buffer silently drops every event on arm64 (the
// 16-byte form only holds on 32-bit). The buffer fits several events so
// one syscall drains the queue.
func (p *fbdevPlatform) readEvents(f *os.File) {
	buf := make([]byte, 512)
	reads := 0
	loggedErrs := map[string]bool{}
	for {
		n, err := f.Read(buf)
		if p.isClosing() {
			return
		}
		reads++
		if reads <= 5 || (err != nil && !loggedErrs[err.Error()]) {
			if err != nil {
				loggedErrs[err.Error()] = true
			}
			logger().Info("fbdev: evdev read outcome",
				"device", f.Name(), "read", reads, "n", n, "err", errString(err))
		}
		if err != nil {
			if err.Error() == "EOF" {
				return
			}
			// O_NONBLOCK with nothing to read — poll briefly and retry.
			time.Sleep(4 * time.Millisecond)
			continue
		}
		for off := 0; off+24 <= n; off += 24 {
			// Field offsets inside the 24-byte event: timeval is 16 bytes
			// on 64-bit, so type/code/value sit at 16/18/20 — reading them
			// at the 32-bit-layout offsets (8/10/12) yields zeroes, which
			// turned every event into a silent EV_SYN.
			typ := binary.LittleEndian.Uint16(buf[off+16 : off+18])
			code := binary.LittleEndian.Uint16(buf[off+18 : off+20])
			value := int32(binary.LittleEndian.Uint32(buf[off+20 : off+24]))
			switch typ {
			case evTypeKey:
				// Trace the first key events of the session: the physical
				// button → evdev code map, including codes that map to
				// game keys and would otherwise stay silent. Autorepeat
				// (value 2) is skipped.
				if value != 2 {
					if n := p.traceKeyEvent(f); n <= 40 {
						logger().Info("fbdev: key event",
							"code", fmt.Sprintf("0x%x (%d)", code, code),
							"value", value)
					}
				}
				p.handleKey(code, value != 0)
			case evTypeAbs:
				p.handleAbs(code, value)
			}
		}
	}
}

// traceKeyEvent counts traced key events per device and reports the
// running count; the caller logs while ≤ 40.
func (p *fbdevPlatform) traceKeyEvent(f *os.File) int {
	name := f.Name()
	p.inputMu.Lock()
	defer p.inputMu.Unlock()
	if p.keyTraces == nil {
		p.keyTraces = make(map[string]int)
	}
	p.keyTraces[name]++
	return p.keyTraces[name]
}

func (p *fbdevPlatform) isClosing() bool {
	p.inputMu.Lock()
	defer p.inputMu.Unlock()
	return p.closing
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// setHeld records press/release of a quit-combo button.
// isHeld reports whether a button is currently in the held map.
func (p *fbdevPlatform) isHeld(code uint16) bool {
	p.inputMu.Lock()
	defer p.inputMu.Unlock()
	_, ok := p.held[code]
	return ok
}

func (p *fbdevPlatform) setHeld(code uint16, down bool) {
	p.inputMu.Lock()
	defer p.inputMu.Unlock()
	if p.held == nil {
		return
	}
	if down {
		if _, ok := p.held[code]; !ok {
			p.held[code] = time.Now()
		}
	} else {
		delete(p.held, code)
	}
}

// logUnknownCode surfaces unmapped evdev codes, once per code on press —
// the diagnostic that reveals what code MENU (or any stray button) really
// sends on a given device.
func (p *fbdevPlatform) logUnknownCode(code uint16, down bool) {
	if !down {
		return
	}
	p.inputMu.Lock()
	seen := p.seenCodes[code]
	p.seenCodes[code] = true
	p.inputMu.Unlock()
	if !seen {
		logger().Info("fbdev: unmapped input code", "code", fmt.Sprintf("0x%x (%d)", code, code))
	}
}

// quitWatcher closes the window once a quit combo has stayed held long
// enough: MENU ≥1.2s, SELECT ≥3s, SELECT+START ≥1.2s, or L1+R1 ≥1.5s.
// Several paths because the MENU button's evdev code is not confirmed on
// this hardware yet, while SELECT/START/L1/R1 mappings are proven. The
// ticker fires even if the release event never arrives.
func (p *fbdevPlatform) quitWatcher() {
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()
	for range t.C {
		if p.isClosing() {
			return
		}
		now := time.Now()
		p.inputMu.Lock()
		via := ""
		if t0, ok := p.held[btnMenu]; ok && now.Sub(t0) >= 3*time.Second {
			via = "MENU(0x138)"
		} else if t0, ok := p.held[btnSel0]; ok && now.Sub(t0) >= 1200*time.Millisecond {
			via = "SELECT(0x162)"
		} else if t0, ok := p.held[btnMode]; ok && now.Sub(t0) >= 1200*time.Millisecond {
			via = "MODE(0x13d)"
		} else if t0, ok := p.held[keyEsc]; ok && now.Sub(t0) >= 3000*time.Millisecond {
			via = "ESC"
		} else if ts, okS := p.held[btnSel0]; okS {
			if tt, okT := p.held[btnStart]; okT &&
				now.Sub(ts) >= 1200*time.Millisecond && now.Sub(tt) >= 1200*time.Millisecond {
				via = "SELECT+START"
			}
		} else if tl, okL := p.held[btnL1]; okL {
			if tr, okR := p.held[btnR1]; okR &&
				now.Sub(tl) >= 1500*time.Millisecond && now.Sub(tr) >= 1500*time.Millisecond {
				via = "L1+R1"
			}
		}
		p.inputMu.Unlock()
		if via != "" {
			logger().Info("fbdev: quit combo held — closing window", "combo", via)
			if w := p.window; w != nil {
				w.Close()
			}
			return
		}
	}
}

func (p *fbdevPlatform) handleKey(code uint16, down bool) {
	var button gpucontext.Buttons
	var key gpucontext.Key
	// While MENU is held, every other key is a system combo (d-pad camera,
	// volume backlight) — restart the quit timer so combos never exit.
	if code != btnMenu {
		p.inputMu.Lock()
		if _, menuHeld := p.held[btnMenu]; menuHeld {
			p.held[btnMenu] = time.Now()
		}
		p.inputMu.Unlock()
	}
	switch code {
	case btnLeft: // real mouse left button
		button = gpucontext.ButtonsLeft
	case btnSouth:
		// A button: a short mouse click (menus and world UI keep working)
		// plus a KeyF13 edge tagging it as a gamepad press — the world
		// layer turns that into attack-nearest / pickup-nearest. The click
		// auto-releases so a held A never turns into held-click walking.
		p.pointerButton(gpucontext.ButtonsLeft, true)
		p.dispatchKey(gpucontext.KeyF13, true)
		go func() {
			time.Sleep(80 * time.Millisecond)
			p.dispatchKey(gpucontext.KeyF13, false)
			p.pointerButton(gpucontext.ButtonsLeft, false)
		}()
	case btnRight, btnEast: // real mouse right / gamepad B
		button = gpucontext.ButtonsRight
	case btnMiddle, btnNorth: // real mouse middle / gamepad X
		button = gpucontext.ButtonsMiddle
	case btnL1:
		key = gpucontext.KeyF1
		p.setHeld(btnL1, down)
	case btnR1:
		key = gpucontext.KeyF2
		p.setHeld(btnR1, down)
	case btnL2:
		key = gpucontext.KeyF3
	case btnR2:
		key = gpucontext.KeyF4
	case btnMenu:
		// Physical MENU (0x138 on this hardware): hold ≥3s quits via the
		// quit watcher (long, because MENU+d-pad is the camera combo and
		// MENU+START captures); while held a KeyF17 modifier tracks the
		// state for the game layer. A short tap fires KeyF15 — the
		// handheld menu overlay.
		p.dispatchKey(gpucontext.KeyF17, down)
		if down {
			// Any other key while MENU is held is a combo (camera, capture,
			// backlight) — restart the quit timer so combos never exit.
			p.inputMu.Lock()
			if _, held := p.held[btnMenu]; held {
				p.held[btnMenu] = time.Now()
			}
			p.inputMu.Unlock()
		}
		p.inputMu.Lock()
		t0, menuHeld := p.held[btnMenu]
		p.inputMu.Unlock()
		p.setHeld(btnMenu, down)
		if !down {
			// Arm the SELECT-echo swallow window first, then handle the tap.
			p.inputMu.Lock()
			p.suppressSel0Until = time.Now().Add(250 * time.Millisecond)
			p.inputMu.Unlock()
			if menuHeld && time.Since(t0) < 800*time.Millisecond {
				p.dispatchKey(gpucontext.KeyF15, true)
				// Release shortly after so the key does not stay held; the
				// press edge lands in the current event batch.
				go func() {
					time.Sleep(120 * time.Millisecond)
					p.dispatchKey(gpucontext.KeyF15, false)
				}()
			}
		}
	case btnSelect:
		key = gpucontext.KeyEscape
		p.setHeld(btnSelect, down)
	case keyVolDown, keyVolUp:
		// Side volume keys: straight through as media keys; the app layer
		// steps the game's BGM+SFX volume.
		if code == keyVolUp {
			key = gpucontext.KeyAudioVolumeUp
		} else {
			key = gpucontext.KeyAudioVolumeDown
		}
	case keyPower:
		// Power cycles the panel: on → dim (backlight off, audio muted,
		// rendering continues) → off (fb blanked too, fully dark) → on.
		// Not a sleep — the process and its sockets stay alive. F14 edges
		// the app to mute, F16 to unmute.
		if down {
			p.advancePowerCycle()
		}
	case btnStart:
		// START is Enter, except while MENU is held — that combo captures
		// the screen instead.
		if down && p.isHeld(btnMenu) {
			p.dispatchKey(gpucontext.KeyPrintScreen, true)
			go func() {
				time.Sleep(120 * time.Millisecond)
				p.dispatchKey(gpucontext.KeyPrintScreen, false)
			}()
			return
		}
		key = gpucontext.KeyEnter
		p.setHeld(btnStart, down)
	case btnMode:
		// Standard BTN_MODE — not this hardware's MENU, kept for safety.
		p.setHeld(btnMode, down)
	case btnSel0:
		// Physical SELECT (0x162 on this hardware) — Escape for the game
		// and a quit-combo button for the watcher. The firmware echoes a
		// SELECT press right after every MENU release; swallow that echo
		// (still tracked as held for the quit combos).
		p.setHeld(btnSel0, down)
		p.inputMu.Lock()
		echo := down && time.Now().Before(p.suppressSel0Until)
		p.inputMu.Unlock()
		if !echo {
			key = gpucontext.KeyEscape
		}
	case btnThumbl:
		key = gpucontext.KeyTab
	case btnThumbr:
		key = gpucontext.KeyBackspace
	default:
		if code < btnLeft {
			key = keyTable[code]
			switch key {
			case gpucontext.KeyLeftShift, gpucontext.KeyRightShift:
				p.inputMu.Lock()
				p.shift = down
				p.inputMu.Unlock()
			case gpucontext.KeyLeftControl, gpucontext.KeyRightControl:
				p.inputMu.Lock()
				p.ctrl = down
				p.inputMu.Unlock()
			case gpucontext.KeyLeftAlt, gpucontext.KeyRightAlt:
				p.inputMu.Lock()
				p.alt = down
				p.inputMu.Unlock()
			}
			// KEY_ESC arrives on the gamepad device itself (its caps list
			// code 1); a 3s hold quits — harmless for normal ESC taps.
			if code == keyEsc {
				p.setHeld(keyEsc, down)
			}
		}
		if key == 0 || key == gpucontext.KeyUnknown {
			p.logUnknownCode(code, down)
		}
	}

	if button != gpucontext.ButtonsNone {
		p.pointerButton(button, down)
		return
	}
	if key != gpucontext.KeyUnknown && key != 0 {
		p.dispatchKey(key, down)
	}
}

func (p *fbdevPlatform) mods() gpucontext.Modifiers {
	var m gpucontext.Modifiers
	if p.shift {
		m |= gpucontext.ModShift
	}
	if p.ctrl {
		m |= gpucontext.ModControl
	}
	if p.alt {
		m |= gpucontext.ModAlt
	}
	return m
}

// advancePowerCycle steps the power-key panel state:
//
//	on  → dim : backlight off (dispdbg setbl 0), audio muted, image keeps
//	            rendering invisibly — the "music screen-off" mode
//	dim → off : additionally blank the framebuffer — fully dark
//	off → on  : unblank and restore the captured backlight level, unmute
//
// Not a sleep: the process, its sockets, and the render loop continue.
func (p *fbdevPlatform) advancePowerCycle() {
	next := (p.powerState + 1) % 3
	switch next {
	case 1: // dim
		if p.powerState == 0 {
			p.captureBacklightLevel()
		}
		p.setPanelBacklight(0)
		p.dispatchKey(gpucontext.KeyF14, true)
		go func() {
			time.Sleep(120 * time.Millisecond)
			p.dispatchKey(gpucontext.KeyF14, false)
		}()
	case 2: // off
		p.setFbBlank(true)
	case 0: // on
		p.setFbBlank(false)
		p.setPanelBacklight(p.dispdbgRestoreLevel)
		p.dispatchKey(gpucontext.KeyF16, true)
		go func() {
			time.Sleep(120 * time.Millisecond)
			p.dispatchKey(gpucontext.KeyF16, false)
		}()
	}
	p.powerState = next
	logger().Info("fbdev: power cycle state", "state", []string{"on", "dim", "off"}[next])
}

// captureBacklightLevel remembers the panel brightness before the first dim
// so the on state can restore it exactly. getbl failed silently on this
// firmware in the field (screen stayed black after the cycle), so a sane
// default is used when the query yields nothing.
func (p *fbdevPlatform) captureBacklightLevel() {
	if p.dispdbgRestoreLevel > 0 {
		return
	}
	const fallbackLevel = 200
	base := "/sys/kernel/debug/dispdbg"
	if _, err := os.Stat(base + "/command"); err != nil {
		p.dispdbgRestoreLevel = fallbackLevel
		return
	}
	for _, w := range [][2]string{
		{base + "/name", "lcd0"},
		{base + "/command", "getbl"},
		{base + "/start", "1"},
	} {
		if err := os.WriteFile(w[0], []byte(w[1]), 0o644); err != nil {
			p.dispdbgRestoreLevel = fallbackLevel
			logger().Warn("fbdev: getbl write failed; backlight restore falls back", "level", fallbackLevel)
			return
		}
	}
	data, err := os.ReadFile(base + "/param")
	if err != nil {
		p.dispdbgRestoreLevel = fallbackLevel
		logger().Warn("fbdev: getbl read failed; backlight restore falls back", "level", fallbackLevel)
		return
	}
	level, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || level <= 0 || level > 255 {
		p.dispdbgRestoreLevel = fallbackLevel
		logger().Warn("fbdev: getbl returned no usable level; backlight restore falls back", "level", fallbackLevel)
		return
	}
	p.dispdbgRestoreLevel = level
	logger().Info("fbdev: captured backlight level", "level", level)
}

// setPanelBacklight writes a brightness level through the Allwinner dispdbg
// interface; no-op when the node is absent.
func (p *fbdevPlatform) setPanelBacklight(level int) {
	base := "/sys/kernel/debug/dispdbg"
	if _, err := os.Stat(base + "/command"); err != nil {
		return
	}
	if level <= 0 {
		level = 0
	}
	for _, w := range [][2]string{
		{base + "/name", "lcd0"},
		{base + "/param", strconv.Itoa(level)},
		{base + "/command", "setbl"},
		{base + "/start", "1"},
	} {
		if err := os.WriteFile(w[0], []byte(w[1]), 0o644); err != nil {
			logger().Warn("fbdev: dispdbg setbl write failed", "file", w[0], "err", err.Error())
			return
		}
	}
}

// setFbBlank blanks or unblanks the framebuffer layer.
func (p *fbdevPlatform) setFbBlank(blank bool) {
	value := byte(0)
	if blank {
		value = 4
	}
	if err := os.WriteFile("/sys/class/graphics/fb0/blank", []byte(fmt.Sprintf("%d", value)), 0o644); err != nil {
		logger().Warn("fbdev: fb blank write failed", "err", err.Error())
	}
}


func (p *fbdevPlatform) dispatchKey(key gpucontext.Key, down bool) {	typ := EventKeyUp
	if down {
		typ = EventKeyDown
	}
	wid := p.windowID()
	mods := p.modsSafely()
	p.push(Event{Type: typ, WindowID: wid, Key: key, Mods: mods})
	if down {
		if ch, ok := keyToChar(key, p.shiftDown()); ok {
			p.push(Event{Type: EventChar, WindowID: wid, Char: ch})
		}
	}
}

func (p *fbdevPlatform) windowID() WindowID {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.window != nil {
		return p.window.id
	}
	return 0
}

func (p *fbdevPlatform) modsSafely() gpucontext.Modifiers {
	p.inputMu.Lock()
	defer p.inputMu.Unlock()
	return p.mods()
}

func (p *fbdevPlatform) shiftDown() bool {
	p.inputMu.Lock()
	defer p.inputMu.Unlock()
	return p.shift
}

func (p *fbdevPlatform) pointerButton(button gpucontext.Buttons, down bool) {
	p.inputMu.Lock()
	x, y := p.cursorX, p.cursorY
	if down {
		p.buttons |= button
	} else {
		p.buttons &^= button
	}
	buttons := p.buttons
	p.inputMu.Unlock()
	sx, sy := p.surfacePoint(x, y)
	x, y = float64(sx), float64(sy)

	btn := gpucontext.ButtonMiddle
	switch button {
	case gpucontext.ButtonsLeft:
		btn = gpucontext.ButtonLeft
	case gpucontext.ButtonsRight:
		btn = gpucontext.ButtonRight
	}
	evType := EventPointerDown
	pointerType := gpucontext.PointerDown
	if !down {
		evType = EventPointerUp
		pointerType = gpucontext.PointerUp
	}
	pressure := float32(0)
	if down {
		pressure = 0.5
	}
	p.push(Event{Type: evType, WindowID: p.windowID(), Pointer: gpucontext.PointerEvent{
		Type:        pointerType,
		PointerID:   1,
		X:           x,
		Y:           y,
		Width:       1,
		Height:      1,
		Pressure:    pressure,
		PointerType: gpucontext.PointerTypeMouse,
		IsPrimary:   true,
		Button:      btn,
		Buttons:     buttons,
		Timestamp:   time.Since(p.started),
	}})
}

func (p *fbdevPlatform) handleAbs(code uint16, value int32) {
	p.inputMu.Lock()
	defer p.inputMu.Unlock()
	switch code {
	case evAbsHat0X:
		if int(value) != p.hatX {
			old := p.hatX
			p.hatX = int(value)
			p.queueHatKeys(old, int(value), gpucontext.KeyLeft, gpucontext.KeyRight)
		}
	case evAbsHat0Y:
		if int(value) != p.hatY {
			old := p.hatY
			p.hatY = int(value)
			p.queueHatKeys(old, int(value), gpucontext.KeyUp, gpucontext.KeyDown)
		}
	default:
		p.axes[code] = value
	}
}

// queueHatKeys posts key events for a d-pad axis change. Caller holds inputMu.
func (p *fbdevPlatform) queueHatKeys(oldVal, newVal int, negKey, posKey gpucontext.Key) {
	// Mirror dispatchKey's event shape, including the window ID — without
	// it the app drops the event on a window that does not exist and the
	// d-pad goes dead everywhere while A/clicks keep working. dispatchKey
	// itself cannot be used here: it re-locks inputMu (deadlock).
	wid := p.windowID()
	mods := p.mods()
	post := func(key gpucontext.Key, down bool) {
		typ := EventKeyUp
		if down {
			typ = EventKeyDown
		}
		p.events.Push(Event{Type: typ, WindowID: wid, Key: key, Mods: mods})
	}
	if (oldVal < 0) != (newVal < 0) {
		post(negKey, newVal < 0)
	}
	if (oldVal > 0) != (newVal > 0) {
		post(posKey, newVal > 0)
	}
	p.wake()
}

// fbScaleFromEnv reads GOGPU_FB_SCALE (1-4); 2 renders at half size and
// lets BlitPixels upscale — roughly 4x cheaper on a software rasterizer.
func fbScaleFromEnv() int {
	n, _ := strconv.Atoi(os.Getenv("GOGPU_FB_SCALE"))
	if n < 1 {
		n = 1
	}
	if n > 4 {
		n = 4
	}
	return n
}

// surfaceSize is the panel size divided by the render scale.
func (p *fbdevPlatform) surfaceSize() (int, int) {
	if p.fbScale <= 1 {
		return p.geo.width, p.geo.height
	}
	w := p.geo.width / p.fbScale
	h := p.geo.height / p.fbScale
	if w < 1 || h < 1 {
		return p.geo.width, p.geo.height
	}
	return w, h
}

// surfacePoint maps framebuffer pixel coordinates into surface space.
func (p *fbdevPlatform) surfacePoint(x, y float64) (float32, float32) {
	if p.fbScale <= 1 {
		return float32(x), float32(y)
	}
	return float32(x / float64(p.fbScale)), float32(y / float64(p.fbScale))
}

// cursorLoop moves the virtual cursor from stick deflection at a fixed
// rate and posts pointer-move events.
func (p *fbdevPlatform) cursorLoop() {
	ticker := time.NewTicker(cursorTick)
	defer ticker.Stop()
	for range ticker.C {
		p.inputMu.Lock()
		if p.closing {
			p.inputMu.Unlock()
			return
		}
		dx := stickDeflection(p.axes[evAbsX]) + stickDeflection(p.axes[evAbsRX])
		dy := stickDeflection(p.axes[evAbsY]) + stickDeflection(p.axes[evAbsRY])
		if dx == 0 && dy == 0 {
			p.inputMu.Unlock()
			continue
		}
		scale := float64(cursorTick) / float64(time.Second)
		p.cursorX += dx * cursorSpeed * scale
		p.cursorY += dy * cursorSpeed * scale
		if p.cursorX < 0 {
			p.cursorX = 0
		}
		if p.cursorY < 0 {
			p.cursorY = 0
		}
		if p.cursorX > float64(p.geo.width-1) {
			p.cursorX = float64(p.geo.width - 1)
		}
		if p.cursorY > float64(p.geo.height-1) {
			p.cursorY = float64(p.geo.height - 1)
		}
		x, y := p.cursorX, p.cursorY
		buttons := p.buttons
		p.inputMu.Unlock()
		sx, sy := p.surfacePoint(x, y)
		p.push(Event{Type: EventPointerMove, WindowID: p.windowID(), Pointer: gpucontext.PointerEvent{
			Type:        gpucontext.PointerMove,
			PointerID:   1,
			X:           float64(sx),
			Y:           float64(sy),
			Width:       1,
			Height:      1,
			PointerType: gpucontext.PointerTypeMouse,
			IsPrimary:   true,
			Button:      gpucontext.ButtonNone,
			Buttons:     buttons,
			Timestamp:   time.Since(p.started),
		}})
	}
}

// stickDeflection normalizes a 0..255 axis to -1..1 with deadzone and a
// cubic response curve for fine aiming.
func stickDeflection(v int32) float64 {
	d := (float64(v) - stickCenter) / stickRange
	if d > 1 {
		d = 1
	} else if d < -1 {
		d = -1
	}
	const dead = 24.0 / stickRange
	if d > -dead && d < dead {
		return 0
	}
	return d * d * d
}

// --- key mapping ---

// keyTable maps Linux KEY_* codes to gpucontext keys. Built once.
var keyTable = buildKeyTable()

func buildKeyTable() map[uint16]gpucontext.Key {
	m := make(map[uint16]gpucontext.Key)
	letters := []uint16{keyQ, keyW, keyE, keyR, keyT, keyY, keyU, keyI, keyO, keyP,
		keyA, keyS, keyD, keyF, keyG, keyH, keyJ, keyK, keyL,
		keyZ, keyX, keyC, keyV, keyB, keyN, keyM}
	letterKeys := []gpucontext.Key{gpucontext.KeyQ, gpucontext.KeyW, gpucontext.KeyE, gpucontext.KeyR, gpucontext.KeyT, gpucontext.KeyY, gpucontext.KeyU, gpucontext.KeyI, gpucontext.KeyO, gpucontext.KeyP,
		gpucontext.KeyA, gpucontext.KeyS, gpucontext.KeyD, gpucontext.KeyF, gpucontext.KeyG, gpucontext.KeyH, gpucontext.KeyJ, gpucontext.KeyK, gpucontext.KeyL,
		gpucontext.KeyZ, gpucontext.KeyX, gpucontext.KeyC, gpucontext.KeyV, gpucontext.KeyB, gpucontext.KeyN, gpucontext.KeyM}
	for i, c := range letters {
		m[c] = letterKeys[i]
	}
	for d := 1; d <= 9; d++ {
		m[key1+uint16(d)-1] = gpucontext.Key0 + gpucontext.Key(d)
	}
	m[key0] = gpucontext.Key0
	for f := 1; f <= 10; f++ {
		m[keyF1+uint16(f)-1] = gpucontext.KeyF1 + gpucontext.Key(f-1)
	}
	m[keyF11] = gpucontext.KeyF11
	m[keyF12] = gpucontext.KeyF12

	m[keyEscape] = gpucontext.KeyEscape
	m[keyBackspace] = gpucontext.KeyBackspace
	m[keyTab] = gpucontext.KeyTab
	m[keyEnter] = gpucontext.KeyEnter
	m[keySpace] = gpucontext.KeySpace
	m[keyInsert] = gpucontext.KeyInsert
	m[keyDelete] = gpucontext.KeyDelete
	m[keyHome] = gpucontext.KeyHome
	m[keyEnd] = gpucontext.KeyEnd
	m[keyPageUp] = gpucontext.KeyPageUp
	m[keyPageDown] = gpucontext.KeyPageDown
	m[keyUp] = gpucontext.KeyUp
	m[keyLeft] = gpucontext.KeyLeft
	m[keyRight] = gpucontext.KeyRight
	m[keyDown] = gpucontext.KeyDown

	m[keyLeftShift] = gpucontext.KeyLeftShift
	m[keyRightShift] = gpucontext.KeyRightShift
	m[keyLeftCtrl] = gpucontext.KeyLeftControl
	m[keyRightCtrl] = gpucontext.KeyRightControl
	m[keyLeftAlt] = gpucontext.KeyLeftAlt
	m[keyRightAlt] = gpucontext.KeyRightAlt

	m[keyMinus] = gpucontext.KeyMinus
	m[keyEqual] = gpucontext.KeyEqual
	m[keyLeftBrace] = gpucontext.KeyLeftBracket
	m[keyRightBrace] = gpucontext.KeyRightBracket
	m[keyBackslash] = gpucontext.KeyBackslash
	m[keySemicolon] = gpucontext.KeySemicolon
	m[keyApostrophe] = gpucontext.KeyApostrophe
	m[keyGrave] = gpucontext.KeyGrave
	m[keyComma] = gpucontext.KeyComma
	m[keyDot] = gpucontext.KeyPeriod
	m[keySlash] = gpucontext.KeySlash
	return m
}

// keyToChar derives printable ASCII for Char events (US layout).
func keyToChar(key gpucontext.Key, shift bool) (rune, bool) {
	if key >= gpucontext.KeyA && key <= gpucontext.KeyZ {
		if shift {
			return rune('A' + (key - gpucontext.KeyA)), true
		}
		return rune('a' + (key - gpucontext.KeyA)), true
	}
	if key >= gpucontext.Key0 && key <= gpucontext.Key9 {
		return rune('0' + (key - gpucontext.Key0)), true
	}
	switch key {
	case gpucontext.KeySpace:
		return ' ', true
	case gpucontext.KeyMinus:
		return pick(shift, '_', '-')
	case gpucontext.KeyEqual:
		return pick(shift, '+', '=')
	case gpucontext.KeyLeftBracket:
		return pick(shift, '{', '[')
	case gpucontext.KeyRightBracket:
		return pick(shift, '}', ']')
	case gpucontext.KeyBackslash:
		return pick(shift, '|', '\\')
	case gpucontext.KeySemicolon:
		return pick(shift, ':', ';')
	case gpucontext.KeyApostrophe:
		return pick(shift, '"', '\'')
	case gpucontext.KeyGrave:
		return pick(shift, '~', '`')
	case gpucontext.KeyComma:
		return pick(shift, '<', ',')
	case gpucontext.KeyPeriod:
		return pick(shift, '>', '.')
	case gpucontext.KeySlash:
		return pick(shift, '?', '/')
	case gpucontext.KeyEnter:
		return '\n', true
	case gpucontext.KeyTab:
		return '\t', true
	}
	return 0, false
}

func pick(shift bool, shifted, plain rune) (rune, bool) {
	if shift {
		return shifted, true
	}
	return plain, true
}
