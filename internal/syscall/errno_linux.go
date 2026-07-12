//go:build linux && (amd64 || arm64)

package syscall

// Link __errno_location from libc.so.6 (glibc and musl both export it).
// On glibc >= 2.34, libc.so.6 is the real library; libdl.so.2 is a stub.
// On musl, libc.so.6 is a symlink. Either way, __errno_location is available.
//
//go:cgo_import_dynamic goffi_errno_location __errno_location "libc.so.6"
//go:cgo_import_dynamic _ _ "libc.so.6"
