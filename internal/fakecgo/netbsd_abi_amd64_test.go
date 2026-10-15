// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build netbsd && !cgo && !goffi_static

package fakecgo

// amd64 defines __CPU_SIMPLE_LOCK_PAD.
const (
	wantMutexSize  = 48
	wantMutexOwner = 16
)
