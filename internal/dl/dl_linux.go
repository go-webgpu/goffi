//go:build linux

// Linux-specific constants for dynamic library loading.
//
// These constants differ from macOS values but the dlopen/dlsym API is identical
// (POSIX standardized). The calling convention is System V AMD64 ABI on both platforms.
//
// Reference: https://codebrowser.dev/glibc/glibc/bits/dlfcn.h.html

package dl

// Link to libdl.so.2 functions using cgo_import_dynamic.
// This works under both CGO_ENABLED=0 (where fakecgo provides the cgo runtime)
// and CGO_ENABLED=1 (where the standard runtime/cgo is linked, see cgo.go).
//
// Note on glibc >= 2.34: libdl.so.2 is a stub (an empty .so with a versioned
// symlink to libc.so.6). dlopen/dlsym/dlerror/dlclose all live in libc.so.6
// itself. We still ask the dynamic linker for "libdl.so.2" because
//   (a) the stub exists on every glibc release shipped with that version, so
//       SONAME-based lookups keep working, and
//   (b) older glibc (< 2.34) and musl still ship the real libdl.so.2.
// Either way, ld.so resolves the symbols via the normal scope rules and the
// caller never has to care which .so they ended up in.

//go:cgo_import_dynamic goffi_dlopen dlopen "libdl.so.2"
//go:cgo_import_dynamic goffi_dlsym dlsym "libdl.so.2"
//go:cgo_import_dynamic goffi_dlerror dlerror "libdl.so.2"
//go:cgo_import_dynamic goffi_dlclose dlclose "libdl.so.2"

// Force dependency on libdl.so.2
//go:cgo_import_dynamic _ _ "libdl.so.2"

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
