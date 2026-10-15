// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build netbsd && !cgo && !goffi_static

package fakecgo

import (
	"testing"
	"unsafe"
)

// Expected NetBSD layouts, computed by gcc from lib/libpthread/pthread_types.h
// and sys/sys/sigtypes.h. The array pairs turn a layout drift into a build
// failure, so a cross-compiled `go vet` or `go test -c` for netbsd catches it
// without a NetBSD machine.
var (
	_ [wantMutexSize - unsafe.Sizeof(pthread_mutex_t{})]byte
	_ [unsafe.Sizeof(pthread_mutex_t{}) - wantMutexSize]byte
	_ [wantMutexOwner - unsafe.Offsetof(pthread_mutex_t{}.ptm_owner)]byte
	_ [unsafe.Offsetof(pthread_mutex_t{}.ptm_owner) - wantMutexOwner]byte
	_ [40 - unsafe.Sizeof(pthread_cond_t{})]byte
	_ [unsafe.Sizeof(pthread_cond_t{}) - 40]byte
	_ [24 - unsafe.Sizeof(stack_t{})]byte
	_ [unsafe.Sizeof(stack_t{}) - 24]byte
	_ [16 - unsafe.Offsetof(stack_t{}.ss_flags)]byte
	_ [unsafe.Offsetof(stack_t{}.ss_flags) - 16]byte
)

func TestNetBSDABI(t *testing.T) {
	tests := []struct {
		name      string
		got, want uintptr
	}{
		{"pthread_mutex_t size", unsafe.Sizeof(pthread_mutex_t{}), wantMutexSize},
		{"pthread_mutex_t ptm_owner offset", unsafe.Offsetof(pthread_mutex_t{}.ptm_owner), wantMutexOwner},
		{"pthread_mutex_t ptm_waiters offset", unsafe.Offsetof(pthread_mutex_t{}.ptm_waiters), wantMutexOwner + 8},
		{"pthread_mutex_t ptm_recursed offset", unsafe.Offsetof(pthread_mutex_t{}.ptm_recursed), wantMutexOwner + 16},
		{"pthread_mutex_t ptm_spare2 offset", unsafe.Offsetof(pthread_mutex_t{}.ptm_spare2), wantMutexOwner + 24},
		{"pthread_cond_t size", unsafe.Sizeof(pthread_cond_t{}), 40},
		{"pthread_cond_t ptc_waiters offset", unsafe.Offsetof(pthread_cond_t{}.ptc_waiters), 8},
		{"pthread_cond_t ptc_mutex offset", unsafe.Offsetof(pthread_cond_t{}.ptc_mutex), 24},
		{"pthread_cond_t ptc_private offset", unsafe.Offsetof(pthread_cond_t{}.ptc_private), 32},
		{"stack_t size", unsafe.Sizeof(stack_t{}), 24},
		{"stack_t ss_size offset", unsafe.Offsetof(stack_t{}.ss_size), 8},
		{"stack_t ss_flags offset", unsafe.Offsetof(stack_t{}.ss_flags), 16},
		{"mutex initializer magic", uintptr(PTHREAD_MUTEX_INITIALIZER.ptm_magic), 0x33330003},
		{"cond initializer magic", uintptr(PTHREAD_COND_INITIALIZER.ptc_magic), 0x55550005},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
		}
	}
}
