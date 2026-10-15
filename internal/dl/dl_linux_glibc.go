//go:build linux && !android && !goffi_static && !goffi_universal && !goffi_musl

// Dynamic symbol imports for Linux/glibc. The musl flavor lives in
// dl_musl_amd64.go / dl_musl_arm64.go behind the goffi_musl build tag.

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
//   (b) older glibc (< 2.34) still ships the real libdl.so.2.
// musl has no libdl.so.2 at all; see dl_musl_<arch>.go.
// Either way, ld.so resolves the symbols via the normal scope rules and the
// caller never has to care which .so they ended up in.

//go:cgo_import_dynamic goffi_dlopen dlopen "libdl.so.2"
//go:cgo_import_dynamic goffi_dlsym dlsym "libdl.so.2"
//go:cgo_import_dynamic goffi_dlerror dlerror "libdl.so.2"
//go:cgo_import_dynamic goffi_dlclose dlclose "libdl.so.2"

// Force dependency on libdl.so.2
//go:cgo_import_dynamic _ _ "libdl.so.2"
