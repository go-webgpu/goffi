//go:build (linux || darwin || freebsd) && amd64

#include "textflag.h"

// goffi_errno_location_stub: JMP to the dynamically linked errno function.
// On Linux/FreeBSD: __errno_location (from libc.so.6 / libc.so.7)
// On macOS: __error (from libSystem.B.dylib)
// In all cases the dynamic symbol is imported as goffi_errno_location.
TEXT goffi_errno_location_stub(SB), NOSPLIT|NOFRAME, $0-0
	JMP goffi_errno_location(SB)
