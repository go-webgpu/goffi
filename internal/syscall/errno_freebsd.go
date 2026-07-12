//go:build freebsd && (amd64 || arm64)

package syscall

// Link __error from libc.so.7 (FreeBSD equivalent of __errno_location).
// __error() returns a pointer to the per-thread errno variable.
//
//go:cgo_import_dynamic goffi_errno_location __error "libc.so.7"
//go:cgo_import_dynamic _ _ "libc.so.7"
