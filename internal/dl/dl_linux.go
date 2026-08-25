//go:build linux && !android && !cgo

// Linux no-cgo implementation: link to libdl.so.2 functions via
// cgo_import_dynamic. The shared RTLD_* constants live in dl_linux_consts.go
// (compiled for both the cgo and !cgo paths).
//
// The dlopen/dlsym API is POSIX-standardized; the calling convention is
// System V AMD64 ABI.
//
// Reference: https://codebrowser.dev/glibc/glibc/bits/dlfcn.h.html

package dl

// Link to libdl.so.2 functions using cgo_import_dynamic.
// This is used with CGO_ENABLED=0, where fakecgo provides the cgo runtime.
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
