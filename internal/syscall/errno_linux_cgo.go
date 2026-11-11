// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build linux && cgo && (amd64 || arm64) && !goffi_static

package syscall

import "github.com/go-webgpu/goffi/internal/dl"

// errnoFnAddr is the errno accessor's address, resolved through cgo once:
// Bionic's __errno on Android, __errno_location on glibc and musl.
var errnoFnAddr = dl.ErrnoAddr()

// ErrnoFnAddr returns the errno accessor's address. A zero result means that
// the platform did not expose the accessor; syscallN then skips capture.
func ErrnoFnAddr() uintptr {
	return errnoFnAddr
}
