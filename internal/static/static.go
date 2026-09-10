// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

// Package static reports whether goffi was built in "static" mode, i.e. with
// every //go:cgo_import_dynamic directive compiled out.
//
// Background: the Go linker marks a binary as dynamically linked (it writes a
// PT_INTERP program header and DT_NEEDED entries) as soon as any package in the
// build carries a //go:cgo_import_dynamic directive. That happens even with
// CGO_ENABLED=0 and internal linking, and -extldflags '-static' cannot undo it
// because no external linker is involved. goffi needs those directives for
// dlopen/dlsym/__errno_location, so any binary that imports goffi is dynamic.
//
// The goffi_static build tag removes those directives. The FFI API keeps its
// shape, but LoadLibrary/GetSymbol/CallFunction return ErrDisabled instead of
// calling into libc, and the resulting binary runs on Alpine, in a scratch
// container, or anywhere else without an ELF interpreter.
//
// The tag is a no-op on Windows (which resolves symbols through kernel32 rather
// than cgo_import_dynamic) and on Android (whose Bionic runtime is always
// dynamically linked).
package static

import "errors"

// ErrDisabled is returned by every entry point that would otherwise need the
// dynamic loader when the goffi_static build tag is set.
var ErrDisabled = errors.New(
	"goffi: FFI is disabled in this build (built with -tags goffi_static); " +
		"rebuild without the tag to load shared libraries")
