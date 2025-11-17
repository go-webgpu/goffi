//go:build linux

package ffi

// fakecgo enables runtime.cgocall without CGO_ENABLED=1
// This allows our complete FFI implementation to work safely.
//
// Status: ✅ FULLY WORKING
// ✅ syscall6 (internal/syscall) - Core C function calls
// ✅ Dlopen/Dlsym/Dlerror (internal/dl) - Library loading
// ✅ Zero external dependencies - complete independence!
//
// See docs/LINUX_FFI_IMPLEMENTATION.md for details.

import (
	_ "github.com/go-webgpu/goffi/internal/fakecgo"
)
