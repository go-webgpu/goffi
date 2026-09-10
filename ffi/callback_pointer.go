//go:build (linux || darwin || freebsd) && (amd64 || arm64) && !goffi_static

package ffi

import "unsafe"

// pointerFromNative reconstructs a pointer from a native callback register or
// stack slot. The native caller owns the memory, so Go's garbage collector does
// not track or move it; the compiler cannot recover that provenance from uintptr.
//
//go:nocheckptr
func pointerFromNative(address uintptr) unsafe.Pointer {
	//nolint:govet,gosec // Native address; see the function contract above.
	return unsafe.Pointer(address)
}
