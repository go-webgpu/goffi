// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors
// SPDX-FileCopyrightText: 2025-2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && (darwin || freebsd || linux || netbsd) && !android && !goffi_static

package fakecgo

import "structs"

type (
	size_t uintptr
	// Sources:
	// Darwin (32 bytes) - https://github.com/apple/darwin-xnu/blob/2ff845c2e033bd0ff64b5b6aa6063a1f8f65aa32/bsd/sys/_types.h#L74
	// FreeBSD (32 bytes) - https://github.com/DoctorWkt/xv6-freebsd/blob/d2a294c2a984baed27676068b15ed9a29b06ab6f/include/signal.h#L98C9-L98C21
	// Linux (128 bytes) - https://github.com/torvalds/linux/blob/ab75170520d4964f3acf8bb1f91d34cbc650688e/arch/x86/include/asm/signal.h#L25
	sigset_t [128]byte
	// pthread_attr_t is opaque: 64 bytes covers every target (glibc 56 on amd64,
	// 64 on arm64; Darwin 64; FreeBSD and NetBSD are smaller). Use uint64
	// elements so it gets the 8-byte alignment of the C type when it lives on
	// the Go stack.
	pthread_attr_t [8]uint64
	pthread_t      int
	pthread_key_t  uint64
)

// for pthread_sigmask:

// sighow is the how argument of pthread_sigmask. Its values differ between
// operating systems (Linux 0/1/2, Darwin and the BSDs 1/2/3), so SIG_BLOCK,
// SIG_UNBLOCK and SIG_SETMASK are defined in the per-OS libcgo_<os>.go files.
type sighow int32

type G struct {
	_       structs.HostLayout
	stacklo uintptr
	stackhi uintptr
}

type ThreadStart struct {
	_   structs.HostLayout
	g   *G
	tls *uintptr
	fn  uintptr
}
