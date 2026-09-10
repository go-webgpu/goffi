// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && linux && !android && !goffi_universal

package fakecgo

// setupUniversalTLS is a no-op outside the goffi_universal build: the default
// and goffi_musl binaries are launched by the host loader, which sets up TLS
// before the Go entry point runs.
//
//go:nosplit
func setupUniversalTLS() {}
