//go:build netbsd && (amd64 || arm64) && !goffi_static

package syscall

// Link __errno from libc.so (the NetBSD equivalent of glibc's
// __errno_location and of FreeBSD/macOS __error). NetBSD's <errno.h> defines
// errno as (*__errno()), so __errno() returns a pointer to the calling
// thread's errno.
//
// Reference: https://github.com/NetBSD/src/blob/trunk/include/errno.h
//
//go:cgo_import_dynamic goffi_errno_location __errno "libc.so"
//go:cgo_import_dynamic _ _ "libc.so"
