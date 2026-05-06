//go:build darwin

// macOS-specific constants for dynamic library loading.
//
// These constants differ from Linux values but the dlopen/dlsym API is identical
// (POSIX standardized). The calling convention is System V AMD64 ABI on both platforms.
//
// Reference: https://opensource.apple.com/source/dyld/dyld-360.14/include/dlfcn.h.auto.html

package dl

// Link to libSystem.B.dylib functions using cgo_import_dynamic.
// This works under both CGO_ENABLED=0 (where fakecgo provides the cgo runtime)
// and CGO_ENABLED=1 (where the standard runtime/cgo is linked, see cgo.go).
//
// On macOS, dlopen/dlsym/dlerror are part of libSystem.B.dylib
// (unlike Linux where they're in libdl.so.2).

//go:cgo_import_dynamic goffi_dlopen dlopen "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic goffi_dlsym dlsym "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic goffi_dlerror dlerror "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic goffi_dlclose dlclose "/usr/lib/libSystem.B.dylib"

// Force dependency on libSystem.B.dylib
//go:cgo_import_dynamic _ _ "/usr/lib/libSystem.B.dylib"

// RTLD constants from <dlfcn.h> for dynamic library loading on macOS.
const (
	// RTLD_LAZY performs relocations at an implementation-dependent time.
	RTLD_LAZY = 0x1

	// RTLD_NOW resolves all symbols when loading the library (recommended).
	RTLD_NOW = 0x2

	// RTLD_LOCAL makes symbols not available for relocation processing by other modules.
	RTLD_LOCAL = 0x4

	// RTLD_GLOBAL makes all symbols available for relocation processing of other modules.
	// NOTE: Different from Linux (0x00100) - macOS uses 0x8
	RTLD_GLOBAL = 0x8
)

// RTLD_DEFAULT is a pseudo-handle for dlsym to search for any loaded symbol.
// NOTE: Different from Linux (0x00000) - macOS uses special value
const RTLD_DEFAULT = 1<<64 - 2 // -2 as uintptr
