//go:build netbsd && !goffi_static

// NetBSD-specific constants for dynamic library loading.
//
// NetBSD ships the dlopen(3) family in libc itself (like FreeBSD, unlike glibc
// which historically split them into libdl.so.2). The RTLD_* values match the
// FreeBSD/Linux numbering; RTLD_DEFAULT is the -2 pseudo handle, same as macOS
// and FreeBSD.
//
// Reference: https://github.com/NetBSD/src/blob/trunk/include/dlfcn.h

package dl

// Link to libc.so functions using cgo_import_dynamic.
// This works under both CGO_ENABLED=0 (where fakecgo provides the cgo runtime)
// and CGO_ENABLED=1 (where the standard runtime/cgo is linked, see cgo.go).
//
// NetBSD keeps dlopen/dlsym/dlclose in libc, and its libc SONAME carries no
// version suffix in the DT_NEEDED entry ("libc.so"), which is what fakecgo's
// symbols_netbsd.go already imports for malloc/pthread.

//go:cgo_import_dynamic goffi_dlopen dlopen "libc.so"
//go:cgo_import_dynamic goffi_dlsym dlsym "libc.so"
//go:cgo_import_dynamic goffi_dlerror dlerror "libc.so"
//go:cgo_import_dynamic goffi_dlclose dlclose "libc.so"

// Force dependency on libc.so
//go:cgo_import_dynamic _ _ "libc.so"

// RTLD constants from <dlfcn.h> for dynamic library loading on NetBSD.
const (
	// RTLD_LAZY performs relocations at an implementation-dependent time.
	RTLD_LAZY = 0x00000001

	// RTLD_NOW resolves all symbols when loading the library (recommended).
	RTLD_NOW = 0x00000002

	// RTLD_GLOBAL makes all symbols available for relocation processing of other modules.
	RTLD_GLOBAL = 0x00000100

	// RTLD_LOCAL makes symbols not available for relocation processing by other modules.
	RTLD_LOCAL = 0x00000000
)

// ptrBits is 32 on 32-bit platforms and 64 on 64-bit ones. Spelling it this way
// keeps RTLD_DEFAULT a valid untyped constant on every GOARCH: a literal
// 1<<64 - 2 would overflow uintptr on a 32-bit build.
const ptrBits = 32 << (^uint(0) >> 63)

// RTLD_DEFAULT is a pseudo-handle for dlsym to search for any loaded symbol.
const RTLD_DEFAULT = 1<<ptrBits - 2 // -2 as uintptr
