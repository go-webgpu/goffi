// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && !goffi_static

package fakecgo

import "structs"

// pthread_mutex_t mirrors struct __pthread_mutex_st on amd64, where
// <machine/types.h> defines __CPU_SIMPLE_LOCK_PAD: ptm_errorcheck and
// ptm_ceiling are each followed by three pad bytes (48 bytes in total).
// Sources:
// https://github.com/NetBSD/src/blob/00b2a11ee704a79ccbcee53927ac9828f981b9d3/lib/libpthread/pthread_types.h#L112
// https://github.com/NetBSD/src/blob/e4d669308f50323b468bf84f84cffeb1a4774f23/sys/arch/amd64/include/types.h#L69
type pthread_mutex_t struct {
	_              structs.HostLayout
	ptm_magic      uint32
	ptm_errorcheck uint8
	ptm_pad1       [3]uint8
	ptm_ceiling    uint8
	ptm_pad2       [3]uint8
	ptm_owner      uintptr
	ptm_waiters    uintptr
	ptm_recursed   uint32
	ptm_spare2     uintptr
}
