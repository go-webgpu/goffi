// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build ((linux && !android) || (darwin && !ios)) && (amd64 || arm64)

package jit

import "syscall"

const supported = true

const chunkHeader = 0

func validCode([]byte) bool { return true }

func allocExec(n int) ([]byte, error) {
	return syscall.Mmap(-1, 0, n, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
}

func sealExec(k *chunk) error {
	if err := flushICache(k.mem); err != nil {
		return err
	}
	return syscall.Mprotect(k.mem, syscall.PROT_READ|syscall.PROT_EXEC)
}
