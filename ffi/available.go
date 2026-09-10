// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

package ffi

import (
	"github.com/go-webgpu/goffi/internal/hostlibc"
	"github.com/go-webgpu/goffi/internal/static"
)

// Available reports whether this build of goffi can load shared libraries and
// call foreign functions.
//
// It returns false in two cases. The binary was built with -tags goffi_static,
// a mode that strips every //go:cgo_import_dynamic directive so the Go linker
// produces a fully static executable (no PT_INTERP, no DT_NEEDED); there
// LoadLibrary, GetSymbol and CallFunction return an error wrapping
// ErrStaticBuild instead of calling into libc. Or the binary is a universal
// ("Profile U") build running on a host whose dynamic loader goffi does not
// recognise, so it never bound a libc at startup; there the same three report
// ErrNoHostLibc. The second case is a property of the machine rather than of
// the build, so it can only be answered at run time -- which is the reason to
// ask this function rather than to reason about build tags.
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
// The goffi_static tag is ignored on Windows and Android, where library loading
// never went through cgo_import_dynamic, so Available reports true there even
// when the tag is set.
func Available() bool {
	return !static.Enabled && !hostlibc.Missing
}
