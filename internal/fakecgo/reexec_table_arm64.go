// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && linux && !android && goffi_universal && arm64

package fakecgo

// Linux/arm64 syscall numbers used by the re-exec bridge. arm64 has no legacy
// open/access/readlink syscalls, so the bridge uses the *at forms everywhere
// (with AT_FDCWD); these numbers match the generic syscall table.
const (
	sysRead        = 63
	sysWrite       = 64
	sysClose       = 57
	sysMmap        = 222
	sysExecve      = 221
	sysReadlinkat  = 78
	sysOpenat      = 56
	sysFaccessat   = 48
	sysGetpid      = 172
	sysMemfdCreate = 279
)

// Host dynamic loader + libc SONAME, per libc flavor, for arm64.
const (
	glibcLoader = "/lib/ld-linux-aarch64.so.1"
	glibcLibc   = "libc.so.6"
	muslLoader  = "/lib/ld-musl-aarch64.so.1"
	muslLibc    = "libc.musl-aarch64.so.1"
)
