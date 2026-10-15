// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build netbsd && !cgo && !goffi_static

package fakecgo

// aarch64 does not define __CPU_SIMPLE_LOCK_PAD.
const (
	wantMutexSize  = 40
	wantMutexOwner = 8
)
