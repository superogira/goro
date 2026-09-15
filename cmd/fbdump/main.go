// Command fbdump converts a raw framebuffer dump (or a live /dev/fb0) into
// a PNG, for developing and debugging the goro fbdev backend. The fbdev
// platform can target a plain file via GOGPU_FB, so a run on any machine
// can be inspected afterwards:
//
//	fbdump -in fb.bin -width 640 -height 480 -out frame.png
//
// Supported layouts: 32bpp little-endian with bitfield offsets from
// -roff/-goff/-boff (default ARGB8888: r=16 g=8 b=0), and 16bpp RGB565.
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	in := flag.String("in", "/dev/fb0", "raw framebuffer file")
	out := flag.String("out", "frame.png", "output PNG path")
	width := flag.Int("width", 640, "framebuffer width")
	height := flag.Int("height", 480, "framebuffer height")
	bpp := flag.Int("bpp", 32, "bits per pixel (32 or 16)")
	stride := flag.Int("stride", 0, "bytes per line (0 = width*bpp/8)")
	roff := flag.Uint("roff", 16, "32bpp only: red bitfield offset")
	goff := flag.Uint("goff", 8, "32bpp only: green bitfield offset")
	boff := flag.Uint("boff", 0, "32bpp only: blue bitfield offset")
	rmax := flag.Uint("rmax", 255, "32bpp only: red max value (255 = full 8 bits)")
	gmax := flag.Uint("gmax", 255, "32bpp only: green max value")
	bmax := flag.Uint("bmax", 255, "32bpp only: blue max value")
	flag.Parse()

	if *stride == 0 {
		*stride = *width * *bpp / 8
	}
	data, err := os.ReadFile(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	img := image.NewRGBA(image.Rect(0, 0, *width, *height))
	scale := func(v, max uint32) uint8 {
		if max == 0 || max == 255 {
			return uint8(v)
		}
		return uint8(v * 255 / max)
	}
	for y := 0; y < *height; y++ {
		for x := 0; x < *width; x++ {
			var c color.RGBA
			switch *bpp {
			case 32:
				off := y**stride + x*4
				if off+4 > len(data) {
					continue
				}
				pix := binary.LittleEndian.Uint32(data[off : off+4])
				c = color.RGBA{
					R: scale((pix>>*roff)&uint32(*rmax), uint32(*rmax)),
					G: scale((pix>>*goff)&uint32(*gmax), uint32(*gmax)),
					B: scale((pix>>*boff)&uint32(*bmax), uint32(*bmax)),
					A: 255,
				}
			case 16:
				off := y**stride + x*2
				if off+2 > len(data) {
					continue
				}
				pix := binary.LittleEndian.Uint16(data[off : off+2])
				c = color.RGBA{
					R: uint8((pix >> 11 & 0x1f) * 255 / 31),
					G: uint8((pix >> 5 & 0x3f) * 255 / 63),
					B: uint8((pix & 0x1f) * 255 / 31),
					A: 255,
				}
			default:
				fmt.Fprintf(os.Stderr, "unsupported bpp %d\n", *bpp)
				os.Exit(1)
			}
			img.SetRGBA(x, y, c)
		}
	}
	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%dx%d)\n", *out, *width, *height)
}
