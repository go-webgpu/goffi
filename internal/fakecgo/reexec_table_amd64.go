// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && linux && !android && goffi_universal && amd64

package fakecgo

// Linux/amd64 syscall numbers used by the re-exec bridge. Only the *at forms
// are used so the same Go logic compiles unchanged on arm64 (see the arm64
// table), where the legacy open/access/readlink numbers do not exist.
const (
	sysRead        = 0
	sysWrite       = 1
	sysClose       = 3
	sysMmap        = 9
	sysExecve      = 59
	sysReadlinkat  = 267
	sysOpenat      = 257
	sysFaccessat   = 269
	sysGetpid      = 39
	sysMemfdCreate = 319
)

// Host dynamic loader + libc SONAME, per libc flavor, for amd64. These are the
// only two ABIs goffi's universal build targets. The SONAMEs are passed bare
// to `<loader> --preload <soname>`; each loader resolves its own libc through
// its default search path (verified on glibc and musl).
const (
	glibcLoader = "/lib64/ld-linux-x86-64.so.2"
	glibcLibc   = "libc.so.6"
	muslLoader  = "/lib/ld-musl-x86_64.so.1"
	muslLibc    = "libc.musl-x86_64.so.1"
)
