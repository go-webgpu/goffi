// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build ((linux && !android) || darwin || freebsd) && goffi_static

// Static build: no libc, hence no errno location to import.

package syscall

// ErrnoFnAddr returns 0 in a static build, which the assembly trampolines
// already treat as "skip errno capture" (the same path Windows takes). No C
// function can be called in this mode anyway, so there is no errno to read.
func ErrnoFnAddr() uintptr {
	return 0
}
