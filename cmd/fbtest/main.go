//go:build linux

// fbtest: a tiny RG35XX display diagnostic.
//
// It answers three questions in one run, logging everything to
// fbtest.log next to the binary:
//
//  1. Do writes to /dev/fb0 stick? Paint the whole fb solid, then read
//     the memory back after a delay — if the color survives, nobody is
//     drawing over us; if it changed, the system overlay is actively
//     redrawing and we catch the current content.
//  2. Does the panel show anything from fb0? Hammer the screen in a
//     loop for 25 seconds, cycling a solid color every 5 seconds
//     (red, green, blue, yellow, white) — a repaint every 100ms beats
//     any single-shot overwrite.
//  3. What does the current vscreeninfo look like, and do the standard
//     mode-setting ioctls (FBIOPUT_VSCREENINFO / FBIOPAN_DISPLAY)
//     accept a plain 640x480 mode?
package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	ioctlGetVarScreeninfo = 0x4600
	ioctlPutVarScreeninfo = 0x4601
	ioctlGetFixScreeninfo = 0x4602
	ioctlPanDisplay       = 0x4606
)

type varScreeninfo struct {
	xres, yres, xresVirtual, yresVirtual, xoffset, yoffset uint32
	bitsPerPixel, grayscale                               uint32
	red, green, blue, transp                              [2]uint32
	nonstd, activate, height, width, accelFlags           uint32
	pixclock, leftMargin, rightMargin, upperMargin, lowerMargin uint32
	hsyncLen, vsyncLen, sync, vmode, rotate, colorspace   uint32
	reserved                                              [4]uint32
}

type fixScreeninfo struct {
	id            [16]byte
	smemStart     uintptr
	smemLen       uint32
	_type         uint16
	typeAux       uint16
	visual        uint16
	xpanstep      uint16
	ypanstep      uint16
	ywrapstep    uint16
	lineLength    uint32
	mmioStart     uintptr
	mmioLen       uint32
	accel         uint16
	capabilities  uint16
	reserved      [2]uint16
}

var logFile *os.File

func logf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(logFile, msg)
	fmt.Fprintln(os.Stderr, msg)
}

func ioctl(fd uintptr, req uintptr, arg unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	var err error
	logFile, err = os.OpenFile(dir+"/fbtest.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open log:", err)
		os.Exit(1)
	}
	defer logFile.Close()
	logf("=== fbtest %s ===", time.Now().Format(time.RFC3339))

	fb, err := os.OpenFile("/dev/fb0", os.O_RDWR, 0)
	if err != nil {
		logf("open /dev/fb0: %v", err)
		os.Exit(1)
	}
	defer fb.Close()

	// --- current modes ---
	var vinfo varScreeninfo
	if err := ioctl(fb.Fd(), ioctlGetVarScreeninfo, unsafe.Pointer(&vinfo)); err != nil {
		logf("FBIOGET_VSCREENINFO: %v", err)
	} else {
		logf("vscreen: %dx%d virtual %dx%d offset +%d+%d bpp=%d rotate=%d vmode=%d activate=%d",
			vinfo.xres, vinfo.yres, vinfo.xresVirtual, vinfo.yresVirtual,
			vinfo.xoffset, vinfo.yoffset, vinfo.bitsPerPixel, vinfo.rotate, vinfo.vmode, vinfo.activate)
	}
	var finfo fixScreeninfo
	if err := ioctl(fb.Fd(), ioctlGetFixScreeninfo, unsafe.Pointer(&finfo)); err != nil {
		logf("FBIOGET_FSCREENINFO: %v", err)
	} else {
		logf("fscreen: id=%q smem=%d line=%d visual=%d type=%d", cstr(finfo.id[:]), finfo.smemLen, finfo.lineLength, finfo.visual, finfo._type)
	}

	lineLen := int(finfo.lineLength)
	if lineLen == 0 {
		lineLen = int(vinfo.xresVirtual) * int(vinfo.bitsPerPixel) / 8
	}
	smemLen := int(finfo.smemLen)
	if smemLen == 0 {
		smemLen = lineLen * int(vinfo.yresVirtual)
	}

	mem, err := syscall.Mmap(int(fb.Fd()), 0, smemLen,
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		logf("mmap(%d): %v", smemLen, err)
		os.Exit(1)
	}
	defer syscall.Munmap(mem)
	logf("mmap ok: %d bytes", len(mem))

	// --- 1. do writes stick? paint red, wait, read back ---
	paint(mem, lineLen, 255, 0, 0, 255, 0, 0) // red everywhere
	time.Sleep(1500 * time.Millisecond)
	sample := []int{0, smemLen / 2, smemLen - 4}
	for _, off := range sample {
		if off >= 0 && off+4 <= len(mem) {
			logf("readback +%d: %02x %02x %02x %02x (painted ff 00 00 ff)", off,
				mem[off], mem[off+1], mem[off+2], mem[off+3])
		}
	}

	// --- 2. hammer the panel for 25s, cycling colors every 5s ---
	palette := [][3]byte{{255, 0, 0}, {0, 255, 0}, {0, 0, 255}, {255, 255, 0}, {255, 255, 255}}
	names := []string{"red", "green", "blue", "yellow", "white"}
	start := time.Now()
	writes := 0
	for elapsed := 0; elapsed < 25; {
		stage := (elapsed / 5) % len(palette)
		c := palette[stage]
		paint(mem, lineLen, c[0], c[1], c[2], c[0], c[1], c[2])
		writes++
		if writes%50 == 1 {
			logf("hammer: %s (write #%d)", names[stage], writes)
		}
		time.Sleep(100 * time.Millisecond)
		elapsed = int(time.Since(start).Seconds())
	}
	logf("hammer done: %d repaints", writes)

	// --- 3. try to (re)assert a plain 640x480 mode ---
	fresh := vinfo
	fresh.xres, fresh.yres = 640, 480
	fresh.xoffset, fresh.yoffset = 0, 0
	if err := ioctl(fb.Fd(), ioctlPutVarScreeninfo, unsafe.Pointer(&fresh)); err != nil {
		logf("FBIOPUT_VSCREENINFO 640x480: %v", err)
	} else {
		logf("FBIOPUT_VSCREENINFO 640x480: ok")
	}
	if err := ioctl(fb.Fd(), ioctlPanDisplay, unsafe.Pointer(&fresh)); err != nil {
		logf("FBIOPAN_DISPLAY: %v", err)
	} else {
		logf("FBIOPAN_DISPLAY: ok")
	}
	// paint one more full red after the modeset
	paint(mem, lineLen, 255, 0, 0, 255, 0, 0)
	time.Sleep(3 * time.Second)
	logf("=== fbtest done — did any color appear on the panel? ===")
}

// paint fills the mapping with a checkerboard of two RGB colors (XRGB
// little-endian bytes: B, G, R, X), quarter-screen blocks so that even
// wrong orientations or halves show something.
func paint(mem []byte, lineLen int, r1, g1, b1, r2, g2, b2 byte) {
	if lineLen <= 0 {
		lineLen = 2560
	}
	for off := 0; off+4 <= len(mem); off += 4 {
		quad := (off/lineLen/120 + (off%lineLen)/(lineLen/4)) % 2
		r, g, b := r1, g1, b1
		if quad == 1 {
			r, g, b = r2, g2, b2
		}
		mem[off] = b
		mem[off+1] = g
		mem[off+2] = r
		mem[off+3] = 0xff
	}
	_ = binary.LittleEndian
}

func cstr(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}
