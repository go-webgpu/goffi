//go:build linux

// Linux-specific constants for dynamic library loading.
//
// These constants differ from macOS values but the dlopen/dlsym API is identical
// (POSIX standardized). The calling convention is System V AMD64 ABI on both platforms.
//
// Reference: https://codebrowser.dev/glibc/glibc/bits/dlfcn.h.html

package dl

// RTLD constants from <dlfcn.h> for dynamic library loading on Linux.
const (
	// RTLD_LAZY performs relocations at an implementation-dependent time.
	RTLD_LAZY = 0x00001

	// RTLD_NOW resolves all symbols when loading the library (recommended).
	RTLD_NOW = 0x00002

	// RTLD_GLOBAL makes all symbols available for relocation processing of other modules.
	// NOTE: Different from macOS (0x8) - Linux uses 0x00100
	RTLD_GLOBAL = 0x00100

	// RTLD_LOCAL makes symbols not available for relocation processing by other modules.
	RTLD_LOCAL = 0x00000
)

// RTLD_DEFAULT is a pseudo-handle for dlsym to search for any loaded symbol.
// NOTE: Different from macOS (1<<64 - 2) - Linux uses 0
const RTLD_DEFAULT = 0x00000
