// Copyright 2025 The GoGPU Authors
// SPDX-License-Identifier: MIT

//go:build linux && !(js && wasm)

package gles

import (
	"fmt"
	"image"
	"os"
	"strings"
	"sync/atomic"
	"unsafe"

	"github.com/gogpu/wgpu/hal"
	"github.com/gogpu/wgpu/hal/gles/egl"
	"github.com/gogpu/wgpu/hal/gles/gl"
)

// Queue implements hal.Queue for OpenGL on Linux.
type Queue struct {
	glCtx           *gl.Context
	eglCtx          *egl.Context
	submissionIndex uint64
	fence           *Fence // signaled at each submit for GPU completion tracking
}

// Submit submits command buffers to the GPU.
// After executing all commands, signals the fence with a GL sync object then
// flushes — the fence must precede flush so PollCompleted sees it.
func (q *Queue) Submit(commandBuffers []hal.CommandBuffer) (uint64, error) {
	for _, cb := range commandBuffers {
		cmdBuf, ok := cb.(*CommandBuffer)
		if !ok {
			return 0, fmt.Errorf("gles: invalid command buffer type")
		}

		// Execute recorded commands with GL error checking.
		for i, cmd := range cmdBuf.commands {
			cmd.Execute(q.glCtx)
			if glErr := q.glCtx.GetError(); glErr != 0 {
				detail := fmt.Sprintf("%T", cmd)
				if vaoCmd, ok := cmd.(*BindVAOCommand); ok {
					detail = fmt.Sprintf("%T{vao=%d}", cmd, vaoCmd.vao)
				}
				hal.Logger().Warn("gles: GL error after command", "error", fmt.Sprintf("0x%x", glErr), "index", i, "command", detail)
			}
		}
	}

	q.submissionIndex++

	// Rust wgpu-hal queue.rs:1915-1921: fence.maintain → fence.signal → gl.flush.
	// FenceSync must be inserted BEFORE Flush so the sync object tracks the
	// commands being flushed. Flushing first would leave the fence un-flushed.
	if q.fence != nil {
		q.fence.Maintain()
		if err := q.fence.Signal(q.submissionIndex); err != nil {
			return 0, err
		}
	}

	q.glCtx.Flush()

	return q.submissionIndex, nil
}

// PollCompleted returns the highest submission index known to be completed.
// Polls pending GL sync objects via glGetSynciv (non-blocking, no flush).
// Safe because Submit() always flushes after inserting the fence — the fence
// is guaranteed to be in the GPU command queue by the time we poll it.
// Maintenance (cleanup of completed sync objects) happens in Submit(), not here
// (matches Rust wgpu-hal device.rs:1564 get_fence_value).
func (q *Queue) PollCompleted() uint64 {
	if q.fence != nil {
		return q.fence.GetLatest()
	}
	return q.submissionIndex
}

// debugBufWrites counts logged buffer writes per kind under
// GOGPU_GLES_DEBUG_CLEAR=1.
var debugBufWrites = map[string]*uint32{
	"uniform": new(uint32),
	"vertex":  new(uint32),
	"index":   new(uint32),
}

// WriteBuffer writes data to a buffer immediately.
func (q *Queue) WriteBuffer(buffer hal.Buffer, offset uint64, data []byte) error {
	buf, ok := buffer.(*Buffer)
	if !ok {
		return fmt.Errorf("gles: WriteBuffer: invalid buffer type")
	}
	if len(data) == 0 {
		return nil
	}

	// Diagnostic: dump the first uniform/vertex/index buffer uploads. The
	// uniform stream was proven sane this way (screen=640x480 floats); the
	// remaining unknown inputs for the black-swapchain mystery are vertex
	// and index data. 640.0f = 0x44000000-style heads confirm float data.
	if os.Getenv("GOGPU_GLES_DEBUG_CLEAR") == "1" {
		kind := ""
		limit := uint32(0)
		switch buf.target {
		case gl.UNIFORM_BUFFER:
			kind, limit = "uniform", 4
		case gl.ARRAY_BUFFER:
			kind, limit = "vertex", 2
		case gl.ELEMENT_ARRAY_BUFFER:
			kind, limit = "index", 2
		}
		if kind != "" {
			seq := atomic.AddUint32(debugBufWrites[kind], 1)
			if seq <= limit {
				hexLen := len(data)
				if hexLen > 48 {
					hexLen = 48
				}
				hal.Logger().Info("gles: buffer write",
					"kind", kind, "seq", seq, "size", len(data), "offset", offset,
					"head", fmt.Sprintf("% x", data[:hexLen]))
			}
		}
	}

	q.glCtx.BindBuffer(buf.target, buf.id)
	q.glCtx.BufferSubData(buf.target, int(offset), len(data), unsafe.Pointer(&data[0]))
	q.glCtx.BindBuffer(buf.target, 0)
	return nil
}

// WriteTexture writes data to a texture immediately.
func (q *Queue) WriteTexture(dst *hal.ImageCopyTexture, data []byte, layout *hal.ImageDataLayout, size *hal.Extent3D) error {
	tex, ok := dst.Texture.(*Texture)
	if !ok {
		return fmt.Errorf("gles: invalid texture type for WriteTexture")
	}

	_, format, dataType := textureFormatToGL(tex.format)
	texelBytes := formatTexelBytes(tex.format)

	q.glCtx.BindTexture(tex.target, tex.id)

	if tex.target == gl.TEXTURE_2D {
		widthBytes := size.Width * texelBytes
		var rowStride uint32
		var offset uint64
		if layout != nil {
			if layout.BytesPerRow > 0 {
				rowStride = layout.BytesPerRow
			}
			offset = layout.Offset
		}
		if rowStride == 0 {
			rowStride = widthBytes
		}
		pixels := data
		if offset > 0 {
			if offset >= uint64(len(data)) {
				return fmt.Errorf("gles: WriteTexture offset %d exceeds data %d", offset, len(data))
			}
			pixels = data[offset:]
		}

		switch {
		case rowStride == widthBytes:
			// Tight rows: only R8 needs the alignment relaxation.
			if texelBytes == 1 {
				q.glCtx.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
			}
			q.glCtx.TexSubImage2D(tex.target, int32(dst.MipLevel),
				int32(dst.Origin.X), int32(dst.Origin.Y),
				int32(size.Width), int32(size.Height), format, dataType,
				unsafe.Pointer(&pixels[0]))
			if texelBytes == 1 {
				q.glCtx.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
			}
		case rowStride%texelBytes == 0:
			// Padded rows (wgpu aligns BytesPerRow; callers may pad further):
			// GL_UNPACK_ROW_LENGTH (ES 3.0 core) makes TexSubImage2D skip the
			// padding. Ignoring the stride skewed every non-aligned texture
			// into horizontal streaks — sprite atlases on the rg35xx.
			q.glCtx.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
			q.glCtx.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(rowStride/texelBytes))
			q.glCtx.TexSubImage2D(tex.target, int32(dst.MipLevel),
				int32(dst.Origin.X), int32(dst.Origin.Y),
				int32(size.Width), int32(size.Height), format, dataType,
				unsafe.Pointer(&pixels[0]))
			q.glCtx.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)
			q.glCtx.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
		default:
			// Unusual stride: repack row by row into a tight buffer.
			tight := make([]byte, int(widthBytes)*int(size.Height))
			for row := uint32(0); row < size.Height; row++ {
				src := uint64(row) * uint64(rowStride)
				if src+uint64(widthBytes) > uint64(len(pixels)) {
					return fmt.Errorf("gles: WriteTexture row %d exceeds data", row)
				}
				copy(tight[int(row)*int(widthBytes):int(row+1)*int(widthBytes)], pixels[src:src+uint64(widthBytes)])
			}
			q.glCtx.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
			q.glCtx.TexSubImage2D(tex.target, int32(dst.MipLevel),
				int32(dst.Origin.X), int32(dst.Origin.Y),
				int32(size.Width), int32(size.Height), format, dataType,
				unsafe.Pointer(&tight[0]))
			q.glCtx.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
		}
	}

	q.glCtx.BindTexture(tex.target, 0)

	hal.Logger().Debug("gles: texture written",
		"format", tex.format,
		"width", size.Width,
		"height", size.Height,
	)

	return nil
}

// Present presents a surface texture to the screen.
//
// Before SwapBuffers, blits the Surface's swapchain offscreen FBO to the
// default framebuffer (FBO 0) with an explicit Y-flip. User render passes
// render upside-down into the swapchain FBO (driven by naga's in-shader
// Y-flip); the blit un-flips for presentation. Mirrors Rust wgpu-hal
// src/gles/egl.rs Surface::present (1280-1308).
//
// damageRects is an optional list of rectangles (physical pixels, top-left
// origin) indicating which surface regions changed this frame. When non-empty
// and EGL_KHR_swap_buffers_with_damage is available, the rects are passed to
// eglSwapBuffersWithDamageKHR as compositor hints. EGL uses bottom-left
// origin, so Y coordinates are flipped here. When the extension is unavailable
// or no rects are provided, the standard eglSwapBuffers path is used.
func (q *Queue) Present(surface hal.Surface, _ hal.SurfaceTexture, damageRects []image.Rectangle) error {
	surf, ok := surface.(*Surface)
	if !ok {
		return fmt.Errorf("gles: invalid surface type")
	}

	surf.blitSwapchainToDefault()

	// Diagnostic: full swapchain-FBO scan at frames 1 and 120 (GOGPU_GLES_DEBUG_CLEAR=1).
	// Counts non-black pixels across the whole FBO and samples a 4x4 grid, plus
	// the GL error state after the blit — separates "renders never land in the
	// FBO" (all black) from "content exists but the blit/swap loses it".
	if os.Getenv("GOGPU_GLES_DEBUG_CLEAR") == "1" {
		if n := surf.probeCount.Add(1); n == 1 || n == 120 {
			q.probeSwapchain(surf, n)
		}
	}

	// Diagnostic: magenta-marker mode. The bottom 40px strip is flooded
	// magenta after the content blit — magenta on the panel proves the
	// present path, and whatever shows above the strip is exactly what
	// the swapchain blit delivered this frame.
	if os.Getenv("GOGPU_GLES_DEBUG_CLEAR") == "1" {
		q.glCtx.BindFramebuffer(gl.FRAMEBUFFER, 0)
		q.glCtx.Enable(gl.SCISSOR_TEST)
		q.glCtx.Scissor(0, 0, int32(surf.fboWidth), 40)
		q.glCtx.ClearColor(1, 0, 1, 1)
		q.glCtx.Clear(gl.COLOR_BUFFER_BIT)
		q.glCtx.Disable(gl.SCISSOR_TEST)
	}

	// Use damage-aware swap when the extension is available and rects provided.
	if len(damageRects) > 0 && egl.HasSwapBuffersWithDamage() {
		// Convert image.Rectangle (top-left origin) to EGL packed int32 array
		// (bottom-left origin). Each rect is {x, y, width, height}.
		// Stack-allocate for up to 8 rects (8 * 4 = 32 ints).
		var stackInts [32]int32
		ints := stackInts[:0]
		surfaceHeight := int32(surf.fboHeight)
		for _, r := range damageRects {
			// Y-flip: EGL uses bottom-left origin.
			// egl_y = surface_height - rect.Max.Y
			ints = append(ints,
				int32(r.Min.X),
				surfaceHeight-int32(r.Max.Y),
				int32(r.Dx()),
				int32(r.Dy()),
			)
		}
		result := egl.SwapBuffersWithDamage(surf.eglDisplay, surf.eglSurface, &ints[0], int32(len(damageRects)))
		if result == egl.False {
			return fmt.Errorf("gles: eglSwapBuffersWithDamageKHR failed: error 0x%x", egl.GetError())
		}
		return nil
	}

	// Standard full-surface swap.
	result := egl.SwapBuffers(surf.eglDisplay, surf.eglSurface)
	if result == egl.False {
		return fmt.Errorf("gles: eglSwapBuffers failed: error 0x%x", egl.GetError())
	}

	return nil
}

// probeSwapchain reads back the whole swapchain FBO once and reports how
// much of it is non-black, a 4x4 pixel grid across the surface, and the
// pending GL error (which at this point includes any error the blit left).
// It also dumps the live rasterizer state — a zero viewport or a stale
// scissor box/writemask is exactly the kind of thing that leaves a bound
// FBO untouched while draws are still issued.
func (q *Queue) probeSwapchain(surf *Surface, frame uint32) {
	// pname constants absent from the gl package's const set.
	const (
		pnViewport       = 0x0BA2
		pnScissorBox     = 0x0C10
		pnScissorTest    = 0x0C11
		pnColorWritemask = 0x0C23
		pnDrawFBO        = 0x8CA6
	)
	w, h := int(surf.fboWidth), int(surf.fboHeight)
	if w <= 0 || h <= 0 {
		return
	}
	glErr := q.glCtx.GetError()

	var vp [4]int32
	q.glCtx.GetIntegerv(pnViewport, &vp[0])
	var sb [4]int32
	q.glCtx.GetIntegerv(pnScissorBox, &sb[0])
	var st int32
	q.glCtx.GetIntegerv(pnScissorTest, &st)
	var wm [4]int32
	q.glCtx.GetIntegerv(pnColorWritemask, &wm[0])
	var curFBO int32
	q.glCtx.GetIntegerv(pnDrawFBO, &curFBO)

	q.glCtx.BindFramebuffer(gl.READ_FRAMEBUFFER, surf.swapchainFBO)
	fbStatus := q.glCtx.CheckFramebufferStatus(gl.FRAMEBUFFER)
	buf := make([]byte, w*h*4)
	q.glCtx.ReadPixels(0, 0, int32(w), int32(h), gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&buf[0]))
	readErr := q.glCtx.GetError()
	nonBlack := 0
	nonGreen := 0
	for i := 0; i < len(buf); i += 4 {
		if buf[i] != 0 || buf[i+1] != 0 || buf[i+2] != 0 {
			nonBlack++
			if !(buf[i] == 0 && buf[i+1] == 128 && buf[i+2] == 0) {
				nonGreen++
			}
		}
	}
	var grid strings.Builder
	for gy := 0; gy < 4; gy++ {
		for gx := 0; gx < 4; gx++ {
			x := w * (2*gx + 1) / 8
			y := h * (2*gy + 1) / 8
			o := (y*w + x) * 4
			fmt.Fprintf(&grid, "(%d,%d)=%02x%02x%02x ", x, y, buf[o], buf[o+1], buf[o+2])
		}
	}
	hal.Logger().Info("gles: swapchain probe",
		"frame", frame,
		"glErrorAfterBlit", fmt.Sprintf("0x%x", glErr),
		"glErrorAfterRead", fmt.Sprintf("0x%x", readErr),
		"nonBlack", nonBlack,
		"nonGreen", nonGreen,
		"total", w*h,
		"viewport", fmt.Sprintf("%dx%d+%d+%d", vp[2], vp[3], vp[0], vp[1]),
		"scissor", fmt.Sprintf("%dx%d+%d+%d", sb[2], sb[3], sb[0], sb[1]),
		"scissorTest", st != 0,
		"colorWritemask", fmt.Sprintf("r%d g%d b%d a%d", wm[0], wm[1], wm[2], wm[3]),
		"fboStatus", fmt.Sprintf("0x%x", fbStatus),
		"grid", grid.String())
	q.glCtx.BindFramebuffer(gl.READ_FRAMEBUFFER, 0)
}

// GetTimestampPeriod returns the timestamp period in nanoseconds.
func (q *Queue) GetTimestampPeriod() float32 {
	// OpenGL doesn't have a standard way to query this
	// Return 1.0 to indicate nanoseconds
	return 1.0
}

// SupportsCommandBufferCopies returns false for GLES on Linux.
// GLES uses direct GL calls for writes, not command buffer copy operations.
func (q *Queue) SupportsCommandBufferCopies() bool {
	return false
}

// SetSwapchainSuppressed is a no-op on GLES.
// GLES uses eglSwapBuffers for presentation, which is not affected by command
// submission ordering. See BUG-WGPU-VK-005 (Vulkan-specific).
func (q *Queue) SetSwapchainSuppressed(_ bool) {}
