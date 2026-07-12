//go:build (linux || darwin || freebsd) && (amd64 || arm64)

package syscall

import "unsafe"

// goffi_errno_location_stub is the JMP trampoline that forwards calls to the
// dynamically linked __errno_location (Linux/FreeBSD) or __error (macOS).
// The symbol is defined in errno_stubs_amd64.s / errno_stubs_arm64.s and
// jumps to the goffi_errno_location dynamic symbol imported via
// //go:cgo_import_dynamic in errno_linux.go / errno_darwin.go / errno_freebsd.go.
//
//go:linkname goffi_errno_location_stub goffi_errno_location_stub
var goffi_errno_location_stub byte

// errnoFnABI0 holds the ABI0 address of the errno function stub, set at init time.
// unsafe.Pointer is required here: we are computing the address of an assembly
// trampoline (not a Go heap object) and storing it as a uintptr for later use
// as a C function pointer inside the assembly trampoline in syscallN.
var errnoFnABI0 = uintptr(unsafe.Pointer(&goffi_errno_location_stub)) //nolint:govet // unsafe.Pointer-to-uintptr is intentional: this is an assembly stub address, not a GC-managed pointer.

// ErrnoFnAddr returns the address of the platform's errno-location function
// (__errno_location on Linux/FreeBSD, __error on macOS). This address is
// passed to CallNFloatErrno to enable in-trampoline errno capture.
//
// Returns 0 if errno capture is not supported on the current platform.
func ErrnoFnAddr() uintptr {
	return errnoFnABI0
}
