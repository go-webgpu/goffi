// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build ((linux && !android) || darwin || freebsd) && goffi_static

// Static build: no dynamic loader.
//
// This file replaces dl_unix.go when the goffi_static build tag is set. Without
// the //go:cgo_import_dynamic directives there is no dlopen to jump to, so the
// three entry points fail cleanly instead of trapping in assembly. The
// signatures match dl_unix.go exactly, so every caller still compiles.

package dl

import "github.com/go-webgpu/goffi/internal/static"

// Dlopen always fails in a static build.
func Dlopen(path string, mode int) (uintptr, error) {
	_, _ = path, mode
	return 0, static.ErrDisabled
}

// Dlsym always fails in a static build.
func Dlsym(handle uintptr, name string) (uintptr, error) {
	_, _ = handle, name
	return 0, static.ErrDisabled
}

// Dlclose is a no-op in a static build: no handle can have been opened.
func Dlclose(handle uintptr) error {
	_ = handle
	return nil
}
