// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && linux && !android && !goffi_universal

package fakecgo

// maybeReexecUniversal is a no-op outside the goffi_universal build. The
// default (glibc) and goffi_musl binaries name their libc via DT_NEEDED and
// are launched normally by the host loader, so there is nothing to bridge.
//
//go:nosplit
func maybeReexecUniversal() {}
