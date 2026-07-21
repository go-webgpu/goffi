// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build !cgo && android && arm64

package fakecgo

import "structs"

// These layouts mirror Android NDK LP64 pthread_types.h.  They intentionally
// do not reuse the glibc Linux definitions: Android's pthread_attr_t is 56
// bytes and pthread_mutex_t is 40 bytes on arm64, while pthread_key_t is a
// 32-bit int and pthread_t is a C long.
type (
	size_t uintptr
	// Bionic LP64 uses the asm-generic sigset_t typedef: one unsigned long.
	// Keep its natural 8-byte alignment because pthread_sigmask receives this
	// object by pointer and the ABI probe asserts the native layout.
	sigset_t uint64
	// Use word arrays to preserve Bionic's native alignment as well as size.
	// pthread_attr_t contains pointers/size_t and is 8-byte aligned on LP64.
	pthread_attr_t [7]uint64
	pthread_t      int64
	pthread_key_t  int32

	// bionic's stack_t is struct { void *ss_sp; int ss_flags; size_t ss_size; }
	stack_t struct {
		ss_sp    uintptr
		ss_flags int32
		_pad     int32
		ss_size  size_t
	}

	pthread_cond_t  [12]uint32 // Bionic's int32[12], align 4.
	pthread_mutex_t [10]uint32 // Bionic's int32[10], align 4.
)

var (
	PTHREAD_COND_INITIALIZER  = pthread_cond_t{}
	PTHREAD_MUTEX_INITIALIZER = pthread_mutex_t{}
)

type sighow int32

const (
	SIG_BLOCK   sighow = 0
	SIG_UNBLOCK sighow = 1
	SIG_SETMASK sighow = 2
)

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
