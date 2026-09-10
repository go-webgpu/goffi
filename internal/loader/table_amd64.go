// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build linux && amd64

package loader

// Host loader + libc SONAME per libc flavor for linux/amd64. Keep in sync with
// internal/fakecgo/reexec_table_amd64.go (asserted by a test).
var (
	Glibc = Entry{Loader: "/lib64/ld-linux-x86-64.so.2", LibC: "libc.so.6", Kind: KindGlibc}
	Musl  = Entry{Loader: "/lib/ld-musl-x86_64.so.1", LibC: "libc.musl-x86_64.so.1", Kind: KindMusl}
	Known = true
)
