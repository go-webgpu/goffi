// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build android && arm64 && !cgo

package fakecgo

import (
	"testing"
	"unsafe"
)

// These compile-time array pairs turn an ABI drift into a build failure. The
// runtime assertions below keep the expected layout visible in test output
// when a physical Android runner is available.
var (
	_ [56 - unsafe.Sizeof(pthread_attr_t{})]byte
	_ [unsafe.Sizeof(pthread_attr_t{}) - 56]byte
	_ [48 - unsafe.Sizeof(pthread_cond_t{})]byte
	_ [unsafe.Sizeof(pthread_cond_t{}) - 48]byte
	_ [40 - unsafe.Sizeof(pthread_mutex_t{})]byte
	_ [unsafe.Sizeof(pthread_mutex_t{}) - 40]byte
	_ [8 - unsafe.Sizeof(sigset_t(0))]byte
	_ [unsafe.Sizeof(sigset_t(0)) - 8]byte
	_ [24 - unsafe.Sizeof(stack_t{})]byte
	_ [unsafe.Sizeof(stack_t{}) - 24]byte
)

func TestAndroidBionicABI(t *testing.T) {
	tests := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"pthread_attr_t size", unsafe.Sizeof(pthread_attr_t{}), 56},
		{"pthread_attr_t align", unsafe.Alignof(pthread_attr_t{}), 8},
		{"pthread_cond_t size", unsafe.Sizeof(pthread_cond_t{}), 48},
		{"pthread_cond_t align", unsafe.Alignof(pthread_cond_t{}), 4},
		{"pthread_mutex_t size", unsafe.Sizeof(pthread_mutex_t{}), 40},
		{"pthread_mutex_t align", unsafe.Alignof(pthread_mutex_t{}), 4},
		{"sigset_t size", unsafe.Sizeof(sigset_t(0)), 8},
		{"sigset_t align", unsafe.Alignof(sigset_t(0)), 8},
		{"stack_t size", unsafe.Sizeof(stack_t{}), 24},
		{"stack_t ss_size offset", unsafe.Offsetof(stack_t{}.ss_size), 16},
		{"pthread_t size", unsafe.Sizeof(pthread_t(0)), 8},
		{"pthread_key_t size", unsafe.Sizeof(pthread_key_t(0)), 4},
		{"tls_g offset", androidTLSGOffset, 16},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %d, want %d", tt.got, tt.want)
			}
		})
	}
}
