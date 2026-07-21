// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build android && cgo && arm64

package syscall

import "github.com/go-webgpu/goffi/internal/dl"

// ErrnoFnAddr returns Bionic's __errno function address. A zero result means
// that the platform did not expose the accessor; syscallN then skips capture.
func ErrnoFnAddr() uintptr {
	return dl.AndroidErrnoAddr()
}
