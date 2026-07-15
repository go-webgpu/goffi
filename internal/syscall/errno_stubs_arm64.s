//go:build ((linux && !android) || (android && !cgo) || darwin || freebsd) && arm64

#include "textflag.h"

// goffi_errno_location_stub: B to the dynamically linked errno function.
// On glibc Linux: __errno_location (from libc.so.6); on Android: __errno
// (from libc.so); on FreeBSD: __error (from libc.so.7).
// On macOS: __error (from libSystem.B.dylib)
// In all cases the dynamic symbol is imported as goffi_errno_location.
TEXT goffi_errno_location_stub(SB), NOSPLIT|NOFRAME, $0-0
	B goffi_errno_location(SB)
