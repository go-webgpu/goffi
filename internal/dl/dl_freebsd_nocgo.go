//go:build freebsd && !cgo

package dl

// Link to libc.so.7 functions using cgo_import_dynamic.
// On FreeBSD, dlopen/dlsym/dlclose are part of libc directly
// (unlike Linux where they're in a separate libdl.so.2).

//go:cgo_import_dynamic goffi_dlopen dlopen "libc.so.7"
//go:cgo_import_dynamic goffi_dlsym dlsym "libc.so.7"
//go:cgo_import_dynamic goffi_dlerror dlerror "libc.so.7"
//go:cgo_import_dynamic goffi_dlclose dlclose "libc.so.7"

// Force dependency on libc.so.7
//go:cgo_import_dynamic _ _ "libc.so.7"
