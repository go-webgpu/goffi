// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !ios

package jit

import (
	"sync"
	"unsafe"

	"github.com/go-webgpu/goffi/internal/dl"
)

//go:linkname syscall_syscall syscall.syscall
func syscall_syscall(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr)

var icacheInvalidate = sync.OnceValues(func() (uintptr, error) {
	return dl.Dlsym(^uintptr(1), "sys_icache_invalidate") // RTLD_DEFAULT
})

// flushICache makes the instructions written to b visible to instruction
// fetch with sys_icache_invalidate: macOS does not let user code read the
// cache line sizes from CTR_EL0, which the Linux sequence needs.
func flushICache(b []byte) error {
	fn, err := icacheInvalidate()
	if err != nil {
		return err
	}
	syscall_syscall(fn, uintptr(unsafe.Pointer(unsafe.SliceData(b))), uintptr(len(b)), 0)
	return nil
}
