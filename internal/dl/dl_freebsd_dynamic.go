//go:build freebsd && !goffi_static

// Dynamic symbol imports for FreeBSD. Kept separate from the RTLD_* constants
// so that the goffi_static build tag can drop them; see dl_linux_dynamic.go.

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
