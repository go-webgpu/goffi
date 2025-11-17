//go:build linux && amd64

package runtime

import (
	"unsafe"
)

// Import runtime.asmcgocall - this is the CORRECT way to call C on Linux!
// This works WITHOUT CGO_ENABLED=1
//
//go:linkname asmcgocall runtime.asmcgocall
func asmcgocall(fn, arg unsafe.Pointer) int32

// SyscallN calls a C function using runtime.asmcgocall
// This is the ONLY safe way to call C code on Linux from Go!
func SyscallN(fn uintptr, args ...uintptr) (r1 uintptr, err error) {
	// Create argument structure
	type callArgs struct {
		fn uintptr
		a1 uintptr
		a2 uintptr
		a3 uintptr
		a4 uintptr
		a5 uintptr
		a6 uintptr
		r1 uintptr
	}

	var cargs callArgs
	cargs.fn = fn
	if len(args) > 0 {
		cargs.a1 = args[0]
	}
	if len(args) > 1 {
		cargs.a2 = args[1]
	}
	if len(args) > 2 {
		cargs.a3 = args[2]
	}
	if len(args) > 3 {
		cargs.a4 = args[3]
	}
	if len(args) > 4 {
		cargs.a5 = args[4]
	}
	if len(args) > 5 {
		cargs.a6 = args[5]
	}

	// Call via runtime.asmcgocall
	// We need an assembly stub that will be called by asmcgocall
	asmcgocall(unsafe.Pointer(&syscallStubABI0), unsafe.Pointer(&cargs))

	return cargs.r1, nil
}

// syscallStub will be implemented in assembly
//
//nolint:unused // Called from assembly (syscall_linux_amd64.s)
func syscallStub(args unsafe.Pointer) uint64

// Get address of syscallStub
//
//go:linkname __syscallStub syscallStub
var __syscallStub byte
var syscallStubABI0 = uintptr(unsafe.Pointer(&__syscallStub))
