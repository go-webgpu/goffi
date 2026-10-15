// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build linux && !android && !goffi_static && goffi_universal

// Dynamic symbol imports for the portable "universal" Linux build.
//
// Unlike the default (glibc) and goffi_musl flavors, this file imports the
// dl* symbols with an EMPTY library name. An empty //go:cgo_import_dynamic
// remote library produces an *undefined* dynamic symbol with no DT_NEEDED
// entry, so the resulting binary names no libc SONAME at all -- it works
// against glibc's libc.so.6 and musl's libc.musl-<arch>.so.1 alike.
//
// A binary with undefined dynamic symbols and no DT_NEEDED cannot resolve
// those symbols on its own: something has to map a libc into the process and
// bind them. goffi does that by re-executing itself, very early, through the
// host's own dynamic loader with the host libc pre-loaded -- see
// internal/fakecgo/reexec_universal_linux.go and docs/PROFILE_U.md. After the
// re-exec every symbol below binds from whichever libc the host actually has.
//
// The ELF interpreter is the other half of the story. The Go linker still
// writes the default glibc PT_INTERP, which does not exist on a musl-only
// system, so a universal binary is post-processed to drop the interpreter
// (cmd/goffi-strip-interp / scripts/build-universal.sh). With no interpreter
// the kernel loads the binary directly on every distribution, the Go runtime
// starts, and the re-exec bridge brings up libc before any FFI call.

package dl

//go:cgo_import_dynamic goffi_dlopen dlopen ""
//go:cgo_import_dynamic goffi_dlsym dlsym ""
//go:cgo_import_dynamic goffi_dlerror dlerror ""
//go:cgo_import_dynamic goffi_dlclose dlclose ""

// NOTE: deliberately no `//go:cgo_import_dynamic _ _ "lib..."` force-line here.
// That line is what makes the linker emit a DT_NEEDED entry in the glibc and
// musl flavors; omitting it is precisely what keeps this binary libc-agnostic.
