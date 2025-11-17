//go:build darwin && !cgo

package dl

// Link to libSystem.B.dylib functions using cgo_import_dynamic.
// This tells the Go linker to dynamically link these symbols.
//
// On macOS, dlopen/dlsym/dlerror are part of libSystem.B.dylib
// (unlike Linux where they're in libdl.so.2).
//
// Reference: https://opensource.apple.com/source/dyld/dyld-360.14/include/dlfcn.h.auto.html

//go:cgo_import_dynamic goffi_dlopen dlopen "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic goffi_dlsym dlsym "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic goffi_dlerror dlerror "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic goffi_dlclose dlclose "/usr/lib/libSystem.B.dylib"

// Force dependency on libSystem.B.dylib
//go:cgo_import_dynamic _ _ "/usr/lib/libSystem.B.dylib"
