// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build linux && arm64

package loader

// Host loader + libc SONAME per libc flavor for linux/arm64. Keep in sync with
// internal/fakecgo/reexec_table_arm64.go (asserted by a test).
var (
	Glibc = Entry{Loader: "/lib/ld-linux-aarch64.so.1", LibC: "libc.so.6", Kind: KindGlibc}
	Musl  = Entry{Loader: "/lib/ld-musl-aarch64.so.1", LibC: "libc.musl-aarch64.so.1", Kind: KindMusl}
	Known = true
)
