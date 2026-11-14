// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !android

package jit

import "unsafe"

// flushICache makes the instructions written to b visible to instruction
// fetch on every core: arm64 caches instructions separately from data.
func flushICache(b []byte) error {
	start := uintptr(unsafe.Pointer(unsafe.SliceData(b)))
	flushICacheRange(start, start+uintptr(len(b)))
	return nil
}

// flushICacheRange is in icache_linux_arm64.s.
//
//go:noescape
func flushICacheRange(start, end uintptr)
