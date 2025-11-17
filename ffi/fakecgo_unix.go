//go:build (linux || darwin) && !cgo

package ffi

// fakecgo enables runtime.cgocall without CGO_ENABLED=1
// This allows our complete FFI implementation to work safely on Unix-like systems.
//
// Status: ✅ FULLY WORKING (Linux, macOS, FreeBSD)
// ✅ syscall6 (internal/syscall) - Core C function calls
// ✅ Dlopen/Dlsym/Dlerror (internal/dl) - Library loading
// ✅ Zero external dependencies - complete independence!
//
// The fakecgo package provides runtime.cgocall implementation without CGO.
// This is REQUIRED for our FFI implementation to work on all Unix platforms.

import (
	_ "github.com/go-webgpu/goffi/internal/fakecgo"
)
