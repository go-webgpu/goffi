//go:build darwin && (amd64 || arm64)

package syscall

// Link __error from libSystem.B.dylib (macOS equivalent of __errno_location).
// __error() returns a pointer to the per-thread errno variable.
//
//go:cgo_import_dynamic goffi_errno_location __error "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic _ _ "/usr/lib/libSystem.B.dylib"
