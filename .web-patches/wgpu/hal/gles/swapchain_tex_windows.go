// Copyright 2025 The GoGPU Authors
// SPDX-License-Identifier: MIT

//go:build windows

package gles

import (
	"unsafe"

	"github.com/gogpu/wgpu/hal/gles/gl"
)

// texImage2DAlloc allocates texture storage with no initial data — the
// windows gl wrapper takes the pixel pointer as an unsafe.Pointer.
func texImage2DAlloc(glCtx *gl.Context, internalFormat uint32, width, height int32, format, dataType uint32) {
	glCtx.TexImage2D(gl.TEXTURE_2D, 0, int32(internalFormat), width, height, 0, format, dataType, unsafe.Pointer(nil))
}
