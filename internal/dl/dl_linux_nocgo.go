//go:build linux && !cgo

package dl

// Link to libdl.so.2 functions using cgo_import_dynamic.
// This tells the Go linker to dynamically link these symbols.

//go:cgo_import_dynamic goffi_dlopen dlopen "libdl.so.2"
//go:cgo_import_dynamic goffi_dlsym dlsym "libdl.so.2"
//go:cgo_import_dynamic goffi_dlerror dlerror "libdl.so.2"
//go:cgo_import_dynamic goffi_dlclose dlclose "libdl.so.2"

// Force dependency on libdl.so.2
//go:cgo_import_dynamic _ _ "libdl.so.2"
