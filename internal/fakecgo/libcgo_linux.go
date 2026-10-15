// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors
// SPDX-FileCopyrightText: 2025-2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && linux && !android && !goffi_static

package fakecgo

type (
	pthread_cond_t  [48]byte
	pthread_mutex_t [48]byte
)

var (
	PTHREAD_COND_INITIALIZER  = pthread_cond_t{}
	PTHREAD_MUTEX_INITIALIZER = pthread_mutex_t{}
)

// Source: https://github.com/torvalds/linux/blob/0fcb05b72344f40806cbe810de8872f1698bf053/include/uapi/asm-generic/signal-defs.h#L71
// (the asm-generic values, used by amd64 and arm64; mips, sparc and alpha differ)
const (
	SIG_BLOCK   sighow = 0
	SIG_UNBLOCK sighow = 1
	SIG_SETMASK sighow = 2
)

type stack_t struct {
	/* not implemented */
}
