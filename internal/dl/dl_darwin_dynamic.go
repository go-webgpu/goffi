//go:build darwin && !goffi_static

// Dynamic symbol imports for macOS. Kept separate from the RTLD_* constants so
// that the goffi_static build tag can drop them; see dl_linux_dynamic.go.

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
