// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && linux && !android && goffi_universal

package fakecgo

// setupUniversalTLS makes the thread pointer usable before the Go code in
// x_cgo_init runs.
//
// When _cgo_init is present, rt0_go skips the runtime's own TLS setup and
// delegates it to _cgo_init (see runtime/asm_amd64.s: the "JZ needtls" branch
// is taken only when _cgo_init is nil). On a universal binary the kernel
// loaded directly -- no interpreter, so no host ld.so ran -- the %fs (amd64) /
// TPIDR_EL0 (arm64) thread pointer is therefore still zero. The Go compiler
// emits a g-register reload from thread-local storage (`MOVQ FS:-8, R14` on
// amd64) after every call to an ABI0 assembly function, and that read faults
// when the thread pointer is unset.
//
// This shim, called as the very first statement of x_cgo_init on the universal
// build, points the thread pointer at a scratch page *only when it is not yet
// set up* (the first launch). The fake g value it exposes is never
// dereferenced before the re-exec bridge calls execve. On the re-executed
// launch the host loader has configured real TLS, so arch_prctl(ARCH_GET_FS)
// / MRS TPIDR_EL0 returns non-zero and the shim does nothing.
//
// Implemented in assembly per architecture; it must not itself touch the g
// register or grow the stack.
func setupUniversalTLS()

// utlsScratch is a scratch slot for arch_prctl(ARCH_GET_FS) on amd64 (arm64
// reads TPIDR_EL0 straight into a register and does not use it).
var utlsScratch uintptr
