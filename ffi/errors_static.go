package ffi

import "errors"

// ErrStaticBuild is returned by LoadLibrary / GetSymbol / FreeLibrary when the
// binary was built with -tags goffi_static. That profile removes all
// //go:cgo_import_dynamic directives so the Go internal linker can emit a
// fully static ELF; dynamic library loading is intentionally unavailable.
//
// Use errors.Is(err, ErrStaticBuild) or unwrap through *LibraryError.
var ErrStaticBuild = errors.New("goffi: dynamic loading unavailable in static build (-tags goffi_static)")
