//go:build linux && !android && (amd64 || arm64) && !goffi_static && !goffi_musl && !goffi_universal

package syscall

// Link __errno_location from glibc's libc.so.6. On glibc >= 2.34 this is
// where dlopen lives too; libdl.so.2 is a stub kept for SONAME lookups.
//
// musl exports __errno_location as well, but under a different SONAME
// (libc.musl-<arch>.so.1 -- there is no libc.so.6 on Alpine), so the musl
// flavor of this file is errno_musl_amd64.go / errno_musl_arm64.go behind
// the goffi_musl build tag.
//
//go:cgo_import_dynamic goffi_errno_location __errno_location "libc.so.6"
//go:cgo_import_dynamic _ _ "libc.so.6"
