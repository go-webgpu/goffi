// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build linux && !android && !goffi_static && goffi_musl && !goffi_universal && amd64

// Dynamic symbol imports for Linux/musl (Alpine and friends).
//
// musl ships the entire POSIX surface -- dlopen, pthreads, libm, errno --
// in one object whose name embeds the architecture: libc.musl-x86_64.so.1.
// There is no libdl.so.2 and no libc.so.6 on a musl system, so the glibc
// directives in dl_linux_dynamic.go make the process fail to start ("Error
// loading shared library libdl.so.2: No such file or directory"). This file
// replaces them under the goffi_musl build tag.
//
// The interpreter is part of the same story: the Go linker defaults
// PT_INTERP to the glibc loader path, which does not exist on Alpine, so
// the binary would die in execve before a single instruction runs. The
// //go:cgo_dynamic_linker directive below bakes the musl loader path into
// every binary built with this tag. The compiler restricts that directive
// to cgo-generated code, so musl builds must relax the check for this one
// package:
//
//	CGO_ENABLED=0 go build -tags goffi_musl \
//	    -gcflags=github.com/go-webgpu/goffi/internal/dl=-std ./...
//
// Forgetting the flag is a loud compile error naming this directive, which
// beats the silent alternative: a binary that carries the wrong interpreter
// and fails at startup with a misleading "no such file" about itself.
//
// Tag interplay: goffi_static wins over goffi_musl -- with both set, all
// dynamic imports are compiled out and FFI is disabled, same as plain
// goffi_static.

package dl

//go:cgo_import_dynamic goffi_dlopen dlopen "libc.musl-x86_64.so.1"
//go:cgo_import_dynamic goffi_dlsym dlsym "libc.musl-x86_64.so.1"
//go:cgo_import_dynamic goffi_dlerror dlerror "libc.musl-x86_64.so.1"
//go:cgo_import_dynamic goffi_dlclose dlclose "libc.musl-x86_64.so.1"

// Force dependency on musl libc
//go:cgo_import_dynamic _ _ "libc.musl-x86_64.so.1"

//go:cgo_dynamic_linker "/lib/ld-musl-x86_64.so.1"
