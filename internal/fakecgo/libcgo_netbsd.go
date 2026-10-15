// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Ebitengine Authors
// SPDX-FileCopyrightText: 2025-2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && !goffi_static

package fakecgo

import "structs"

// NetBSD's pthread_mutex_t and pthread_cond_t are structs whose first field is
// a magic number checked by every libpthread call: a zeroed object makes
// pthread_mutex_lock and pthread_cond_broadcast fail with EINVAL without doing
// anything (or abort under PTHREAD_DIAGASSERT=a). pthread_mutex_t's layout
// depends on __CPU_SIMPLE_LOCK_PAD and is defined in libcgo_netbsd_<arch>.go.

// pthread_cond_t mirrors struct __pthread_cond_st.
// Source: https://github.com/NetBSD/src/blob/00b2a11ee704a79ccbcee53927ac9828f981b9d3/lib/libpthread/pthread_types.h#L170
type pthread_cond_t struct {
	_           structs.HostLayout
	ptc_magic   uint32
	ptc_lock    uint8 // __pthread_spin_t, unused
	ptc_waiters uintptr
	ptc_spare   uintptr
	ptc_mutex   uintptr
	ptc_private uintptr
}

const (
	_PT_MUTEX_MAGIC = 0x33330003
	_PT_COND_MAGIC  = 0x55550005
)

// _PTHREAD_MUTEX_INITIALIZER and _PTHREAD_COND_INITIALIZER set the magic and
// zero everything else (__SIMPLELOCK_UNLOCKED is 0 on amd64 and aarch64).
var (
	PTHREAD_COND_INITIALIZER  = pthread_cond_t{ptc_magic: _PT_COND_MAGIC}
	PTHREAD_MUTEX_INITIALIZER = pthread_mutex_t{ptm_magic: _PT_MUTEX_MAGIC}
)

// Source: https://github.com/NetBSD/src/blob/613e27c65223fd2283b6ed679da1197e12f50e27/sys/sys/signal.h#L242
const (
	SIG_BLOCK   sighow = 1
	SIG_UNBLOCK sighow = 2
	SIG_SETMASK sighow = 3
)

// Unlike Linux, NetBSD puts ss_size before ss_flags.
// Source: https://github.com/NetBSD/src/blob/37caa6a62450429dc22bdf5fd7bc7c998c6db339/sys/sys/sigtypes.h#L114
type stack_t struct {
	_        structs.HostLayout
	ss_sp    uintptr
	ss_size  uintptr
	ss_flags int32
}

// Source: https://github.com/NetBSD/src/blob/613e27c65223fd2283b6ed679da1197e12f50e27/sys/sys/signal.h#L261
const SS_DISABLE = 0x004
