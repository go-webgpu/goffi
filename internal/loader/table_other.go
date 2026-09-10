// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !linux || (!amd64 && !arm64)

package loader

// The universal build targets only linux/amd64 and linux/arm64; elsewhere the
// loader flavor is unknown.
var (
	Glibc = Entry{Kind: KindUnknown}
	Musl  = Entry{Kind: KindUnknown}
	Known = false
)
