// Copyright 2025 The GoGPU Authors
// SPDX-License-Identifier: MIT

//go:build (windows || linux) && !(js && wasm)

package gles

import (
	"fmt"
	"os"

	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu/hal"
	"github.com/gogpu/wgpu/hal/gles/gl"
)

// allocateSwapchainFBO creates a persistent swapchain framebuffer with a
// texture-backed color attachment. The texture (not a renderbuffer) matches
// the offscreen-FBO path that renders correctly on the Mali fbdev stack —
// an RBO attachment rendered solid black there while clears landed, so the
// attachment type is the experiment variable.
// Must be called with the GL context current (caller holds AdapterContext lock).
func allocateSwapchainFBO(glCtx *gl.Context, format gputypes.TextureFormat, width, height uint32) (fbo, colorTex uint32, err error) {
	if glCtx == nil {
		return 0, 0, fmt.Errorf("gles: allocateSwapchainFBO: nil gl context")
	}
	if width == 0 || height == 0 {
		return 0, 0, hal.ErrZeroArea
	}

	internalFormat, glFormat, glType := textureFormatToGL(format)

	colorTex = glCtx.GenTextures(1)
	if colorTex == 0 {
		return 0, 0, fmt.Errorf("gles: glGenTextures returned 0")
	}
	glCtx.BindTexture(gl.TEXTURE_2D, colorTex)
	texImage2DAlloc(glCtx, internalFormat, int32(width), int32(height), glFormat, glType)
	glCtx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	glCtx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	glCtx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	glCtx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	glCtx.BindTexture(gl.TEXTURE_2D, 0)

	fbo = glCtx.GenFramebuffers(1)
	if fbo == 0 {
		glCtx.DeleteTextures(colorTex)
		return 0, 0, fmt.Errorf("gles: glGenFramebuffers returned 0")
	}
	glCtx.BindFramebuffer(gl.FRAMEBUFFER, fbo)
	glCtx.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, colorTex, 0)

	status := glCtx.CheckFramebufferStatus(gl.FRAMEBUFFER)

	// Diagnostic seed clear (GOGPU_GLES_DEBUG_CLEAR=1): flood the fresh FBO
	// with green. The frame-1 probe then separates pass-targeting problems
	// from clipped draws — green surviving means no render pass ever wrote
	// this FBO; black means a pass cleared it but its draws rasterized
	// nothing.
	if os.Getenv("GOGPU_GLES_DEBUG_CLEAR") == "1" {
		glCtx.Disable(gl.SCISSOR_TEST)
		glCtx.ClearColor(0, 0.5, 0, 1)
		glCtx.Clear(gl.COLOR_BUFFER_BIT)
	}

	glCtx.BindFramebuffer(gl.FRAMEBUFFER, 0)

	if status != gl.FRAMEBUFFER_COMPLETE {
		glCtx.DeleteFramebuffers(fbo)
		glCtx.DeleteTextures(colorTex)
		return 0, 0, fmt.Errorf("gles: swapchain framebuffer incomplete (status 0x%x)", status)
	}

	hal.Logger().Debug("gles: allocated swapchain FBO",
		"fbo", fbo,
		"colorTex", colorTex,
		"width", width,
		"height", height,
		"internalFormat", fmt.Sprintf("0x%x", internalFormat),
	)
	return fbo, colorTex, nil
}

// destroySwapchainFBO releases the swapchain framebuffer and its color
// texture. Safe to call with zero handles or nil context.
func destroySwapchainFBO(glCtx *gl.Context, fbo, colorTex uint32) {
	if glCtx == nil {
		return
	}
	if fbo != 0 {
		glCtx.DeleteFramebuffers(fbo)
	}
	if colorTex != 0 {
		glCtx.DeleteTextures(colorTex)
	}
}

// blitSwapchainToDefaultWith performs the present-time Y-flipping blit from
// the Surface's swapchain FBO to the default framebuffer (FBO 0).
// Must be called with GL context current on the user window DC.
//
// Mirrors Rust wgpu-hal/src/gles/egl.rs Surface::present (1280-1308).
func (s *Surface) blitSwapchainToDefaultWith(glCtx *gl.Context) {
	if glCtx == nil || s.swapchainFBO == 0 {
		return
	}
	if s.fboWidth == 0 || s.fboHeight == 0 {
		return
	}

	glCtx.Disable(gl.SCISSOR_TEST)

	glCtx.BindFramebuffer(gl.READ_FRAMEBUFFER, s.swapchainFBO)
	glCtx.BindFramebuffer(gl.DRAW_FRAMEBUFFER, 0)

	w := int32(s.fboWidth)
	h := int32(s.fboHeight)

	glCtx.BlitFramebuffer(
		0, h, w, 0, // source Y-flipped
		0, 0, w, h, // dest normal
		gl.COLOR_BUFFER_BIT, gl.NEAREST,
	)

	glCtx.BindFramebuffer(gl.READ_FRAMEBUFFER, 0)
	glCtx.BindFramebuffer(gl.DRAW_FRAMEBUFFER, 0)
}

// reconfigureSwapchainFBOWith destroys the existing swapchain FBO and
// allocates a new one. Caller must hold the AdapterContext lock.
func (s *Surface) reconfigureSwapchainFBOWith(glCtx *gl.Context, format gputypes.TextureFormat, width, height uint32) error {
	destroySwapchainFBO(glCtx, s.swapchainFBO, s.colorRenderbuffer)
	s.swapchainFBO = 0
	s.colorRenderbuffer = 0
	s.fboWidth = 0
	s.fboHeight = 0

	fbo, colorRbo, err := allocateSwapchainFBO(glCtx, format, width, height)
	if err != nil {
		return err
	}
	s.swapchainFBO = fbo
	s.colorRenderbuffer = colorRbo
	s.fboWidth = width
	s.fboHeight = height
	return nil
}
