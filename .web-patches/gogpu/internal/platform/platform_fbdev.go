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

	btnSouth  = 0x130 // A
	btnEast   = 0x131 // B
	btnNorth  = 0x133 // X
	btnWest   = 0x134 // Y
	btnTL     = 0x135 // L1
	btnTR     = 0x136 // R1
	btnTL2    = 0x137 // L2
	btnTR2    = 0x138 // R2
	btnSelect = 0x139
	btnStart  = 0x13a
	btnThumbl = 0x13b
	btnThumbr = 0x13c

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
	fbScale          int
	buttons          gpucontext.Buttons
	axes             map[uint16]int32
	hatX, hatY       int
	shift, ctrl, alt bool
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
	geo, err := probeFBGeometry(f)
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
	logger().Info("fbdev: framebuffer ready",
		"device", dev, "size", fmt.Sprintf("%dx%d", geo.width, geo.height),
		"virtual", fmt.Sprintf("%dx%d", geo.widthVirtual, geo.heightVirtual),
		"offsets", fmt.Sprintf("+%d+%d", geo.xoffset, geo.yoffset),
		"bpp", geo.bpp, "line", geo.lineLength, "maplen", len(mem))
	if os.Getenv("GOGPU_FB_PROBE") == "color" {
		p.probeBufferColors()
	}
	p.startInput()
	go p.cursorLoop()
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

func probeFBGeometry(f *os.File) (fbGeometry, error) {
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

	var varBuf [160]byte
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
func (w *fbdevWindow) UseHeadlessSurface() bool { return true }

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
	return PrepareFrameResult{
		ScaleFactor:    1,
		PhysicalWidth:  uint32(w.platform.geo.width),  //nolint:gosec // fb size fits u32
		PhysicalHeight: uint32(w.platform.geo.height), //nolint:gosec // fb size fits u32
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
	for y := 0; y < height && y*stepY < geo.height; y++ {
		src := pixels[y*srcRow : y*srcRow+srcRow]
		row := p.fbMem[dstBase+y*stepY*geo.lineLength:]
		switch geo.bpp {
		case 32:
			if stepX == 1 {
				for x := 0; x < width && x < geo.width; x++ {
					r, g, b := src[x*4], src[x*4+1], src[x*4+2]
					if bgra {
						r, b = b, r
					}
					var pix uint32
					if geo.redBits.length > 0 {
						pix |= (uint32(r) >> (8 - geo.redBits.length)) << geo.redBits.offset
						pix |= (uint32(g) >> (8 - geo.greenBits.length)) << geo.greenBits.offset
						pix |= (uint32(b) >> (8 - geo.blueBits.length)) << geo.blueBits.offset
					} else {
						pix = uint32(r)<<16 | uint32(g)<<8 | uint32(b)
					}
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
				var pix uint32
				if geo.redBits.length > 0 {
					pix |= (uint32(r) >> (8 - geo.redBits.length)) << geo.redBits.offset
					pix |= (uint32(g) >> (8 - geo.greenBits.length)) << geo.greenBits.offset
					pix |= (uint32(b) >> (8 - geo.blueBits.length)) << geo.blueBits.offset
				} else {
					pix = uint32(r)<<16 | uint32(g)<<8 | uint32(b)
				}
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
	glob := os.Getenv("GOGPU_INPUT_GLOB")
	if glob == "" {
		glob = "/dev/input/event*"
	}
	devs, _ := filepath.Glob(glob)
	opened := 0
	for _, d := range devs {
		f, err := os.OpenFile(d, os.O_RDONLY|unix.O_NONBLOCK, 0)
		if err != nil {
			continue
		}
		p.inputMu.Lock()
		p.inputs = append(p.inputs, f)
		p.inputMu.Unlock()
		go p.readEvents(f)
		opened++
	}
	if opened > 0 {
		logger().Info("fbdev: listening to input devices", "count", opened)
	}
}

// readEvents parses one evdev device. input_event on 64-bit Linux:
// {i64 sec, i64 usec, u16 type, u16 code, i32 value} = 16 bytes.
func (p *fbdevPlatform) readEvents(f *os.File) {
	var buf [16]byte
	for {
		n, err := f.Read(buf[:])
		if p.isClosing() {
			return
		}
		if err != nil {
			if err.Error() == "EOF" {
				return
			}
			// O_NONBLOCK with nothing to read — poll briefly and retry.
			time.Sleep(4 * time.Millisecond)
			continue
		}
		if n < 16 {
			continue
		}
		typ := binary.LittleEndian.Uint16(buf[8:10])
		code := binary.LittleEndian.Uint16(buf[10:12])
		value := int32(binary.LittleEndian.Uint32(buf[12:16]))
		switch typ {
		case evTypeKey:
			p.handleKey(code, value != 0)
		case evTypeAbs:
			p.handleAbs(code, value)
		}
	}
}

func (p *fbdevPlatform) isClosing() bool {
	p.inputMu.Lock()
	defer p.inputMu.Unlock()
	return p.closing
}

func (p *fbdevPlatform) handleKey(code uint16, down bool) {
	var button gpucontext.Buttons
	var key gpucontext.Key
	switch code {
	case btnLeft, btnSouth: // real mouse left / gamepad A
		button = gpucontext.ButtonsLeft
	case btnRight, btnEast: // real mouse right / gamepad B
		button = gpucontext.ButtonsRight
	case btnMiddle, btnNorth: // real mouse middle / gamepad X
		button = gpucontext.ButtonsMiddle
	case btnWest: // Y
		key = gpucontext.KeySpace
	case btnTL:
		key = gpucontext.KeyF1
	case btnTR:
		key = gpucontext.KeyF2
	case btnTL2:
		key = gpucontext.KeyF3
	case btnTR2:
		key = gpucontext.KeyF4
	case btnSelect:
		key = gpucontext.KeyEscape
	case btnStart:
		key = gpucontext.KeyEnter
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

func (p *fbdevPlatform) dispatchKey(key gpucontext.Key, down bool) {
	typ := EventKeyUp
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
	mods := p.mods()
	post := func(key gpucontext.Key, down bool) {
		typ := EventKeyUp
		if down {
			typ = EventKeyDown
		}
		p.events.Push(Event{Type: typ, Key: key, Mods: mods})
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
