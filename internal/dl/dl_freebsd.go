//go:build freebsd

// FreeBSD-specific constants for dynamic library loading.
//
// FreeBSD uses the same POSIX dlopen/dlsym API as Linux and macOS.
// RTLD_NOW, RTLD_GLOBAL, RTLD_LOCAL, RTLD_LAZY match Linux values.
// RTLD_DEFAULT matches macOS value (not Linux).
//
// Reference: https://github.com/freebsd/freebsd-src/blob/main/include/dlfcn.h

package dl

// Link to libc.so.7 functions using cgo_import_dynamic.
// This works under both CGO_ENABLED=0 (where fakecgo provides the cgo runtime)
// and CGO_ENABLED=1 (where the standard runtime/cgo is linked, see cgo.go).
//
// On FreeBSD, dlopen/dlsym/dlclose are part of libc directly
// (unlike Linux where they're in a separate libdl.so.2).

//go:cgo_import_dynamic goffi_dlopen dlopen "libc.so.7"
//go:cgo_import_dynamic goffi_dlsym dlsym "libc.so.7"
//go:cgo_import_dynamic goffi_dlerror dlerror "libc.so.7"
//go:cgo_import_dynamic goffi_dlclose dlclose "libc.so.7"

// Force dependency on libc.so.7
//go:cgo_import_dynamic _ _ "libc.so.7"

// RTLD constants from <dlfcn.h> for dynamic library loading on FreeBSD.
const (
	// RTLD_LAZY performs relocations at an implementation-dependent time.
	RTLD_LAZY = 0x00001

	// RTLD_NOW resolves all symbols when loading the library (recommended).
	RTLD_NOW = 0x00002

	// RTLD_GLOBAL makes all symbols available for relocation processing of other modules.
	RTLD_GLOBAL = 0x00100

	// RTLD_LOCAL makes symbols not available for relocation processing by other modules.
	RTLD_LOCAL = 0x00000
)

// RTLD_DEFAULT is a pseudo-handle for dlsym to search for any loaded symbol.
// Same value as macOS, different from Linux (0x00000).
const RTLD_DEFAULT = 1<<64 - 2 // -2 as uintptr
