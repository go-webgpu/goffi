// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && !goffi_static

package fakecgo

import "structs"

// pthread_mutex_t mirrors struct __pthread_mutex_st on aarch64, where
// <machine/types.h> does not define __CPU_SIMPLE_LOCK_PAD (40 bytes in total).
// Sources:
// https://github.com/NetBSD/src/blob/00b2a11ee704a79ccbcee53927ac9828f981b9d3/lib/libpthread/pthread_types.h#L112
// https://github.com/NetBSD/src/blob/55b1c8710a9ac7959a235d5253abb8abfb49e1ff/sys/arch/aarch64/include/types.h#L93
type pthread_mutex_t struct {
	_              structs.HostLayout
	ptm_magic      uint32
	ptm_errorcheck uint8
	ptm_ceiling    uint8
	ptm_owner      uintptr
	ptm_waiters    uintptr
	ptm_recursed   uint32
	ptm_spare2     uintptr
}
