// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build ((linux && !android) || darwin || freebsd) && (amd64 || arm64) && goffi_static

// Static build: no callbacks.
//
// The real dispatcher (callback_amd64.s / callback_arm64.s) enters Go through
// crosscall2, which is supplied by internal/fakecgo. That package is not linked
// in a static build, so keeping the dispatcher would fail at link time with
// "relocation target crosscall2 not defined" as soon as anything referenced it.

package ffi

// NewCallback panics in a static build.
//
// It cannot return an error because its signature is fixed by the dynamic
// implementation, and it cannot return 0 either: a null function pointer handed
// to C would crash later, far from the cause. Check Available before creating
// callbacks in code that may be compiled with -tags goffi_static.
func NewCallback(fn any) uintptr {
	_ = fn
	panic(ErrStaticBuild)
}
