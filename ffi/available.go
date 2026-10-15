// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

package ffi

import "github.com/go-webgpu/goffi/internal/hostlibc"

// Available reports whether this build of goffi can load shared libraries and
// call foreign functions. The answer depends on the build mode:
//
//   - -tags goffi_static: always false, a compile-time constant. The build has
//     no //go:cgo_import_dynamic directives, so the linker emits a fully static
//     executable (no PT_INTERP, no DT_NEEDED) and there is no loader to ask.
//     LoadLibrary and GetSymbol return an error wrapping ErrStaticBuild, and
//     a callback created with NewCallback aborts the process if C calls it.
//     This applies on linux, darwin, freebsd and android (amd64/arm64 only
//     for the first three, arm64 for android). Elsewhere the tag has no
//     effect and Available follows the default rule below.
//   - -tags goffi_universal ("Profile U"): decided at run time, once, before
//     main. The binary re-execs itself through the host's dynamic loader with
//     the host libc preloaded. On a host whose loader goffi does not recognise,
//     or where that re-exec cannot be made to work, the process carries on as a
//     pure-Go program and Available is false; the FFI entry points then return
//     ErrNoHostLibc. The same binary can answer differently on two machines.
//   - Every other build (the default, goffi_musl, CGO_ENABLED=1): always true.
//     The loader is a link-time dependency (PT_INTERP and DT_NEEDED on ELF
//     systems, kernel32 on Windows), so a process that reached main has it;
//     nothing is probed at run time.
//
// Callers that have a pure-Go fallback should branch on this at startup rather
// than treating the first LoadLibrary failure as fatal:
//
//	if ffi.Available() {
//	    backend = newAcceleratedBackend()
//	} else {
//	    backend = newPureGoBackend()
//	}
//
// Under goffi_static the call is a constant false, so the compiler drops the
// accelerated branch and everything only it reaches.
func Available() bool {
	return !staticBuild && !hostlibc.Missing
}
