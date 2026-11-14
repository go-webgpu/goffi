// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

package ffi

import (
	"syscall"
	"unsafe"

	"github.com/go-webgpu/goffi/internal/jit"
	gosyscall "github.com/go-webgpu/goffi/internal/syscall"
	"github.com/go-webgpu/goffi/types"
)

//go:linkname cgoNoCallback runtime.cgoNoCallback
func cgoNoCallback(v bool)

// CallFunctionNoCallback is CallFunction for a C function that does not call
// back into Go and does not keep a pointer to Go memory after it returns: the
// contract cgo's "#cgo nocallback" and "#cgo noescape" annotations state.
//
// In exchange, rvalue, avalue and the values they point to do not escape:
// they can live on the goroutine stack, and a call allocates nothing. That
// stack does not move during the call: the runtime never shrinks the stack of
// a goroutine that is in a C call, and only a callback could grow it. A
// function that calls back anyway makes the runtime panic with "function
// marked with #cgo nocallback called back into Go".
func CallFunctionNoCallback(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
) (syscall.Errno, error) {
	call, rvalue, err := checkCall(cif, fn, rvalue, avalue)
	if err != nil {
		return 0, err
	}
	for i, p := range avalue {
		if p == nil {
			return 0, &InvalidCallInterfaceError{Field: "avalue", Reason: "nil argument pointer", Index: i}
		}
	}
	rv := noescape(rvalue)
	av := unsafe.Slice((*unsafe.Pointer)(noescape(unsafe.Pointer(unsafe.SliceData(avalue)))), len(avalue))
	cgoNoCallback(true)
	if call.Bridge != nil {
		if cerrno, ok := jit.Run((*jit.Code)(call.Bridge), fn, rv, av, gosyscall.ErrnoFnAddr()); ok {
			cgoNoCallback(false)
			return syscall.Errno(cerrno), nil
		}
	}
	cerrno, err := executeFunction(call, fn, rv, av)
	cgoNoCallback(false)
	return syscall.Errno(cerrno), err
}

// noescape hides p from escape analysis, as internal/abi.NoEscape does. It
// reads the integer back through a pointer instead of converting it, a
// conversion go vet reports as a possible misuse of unsafe.Pointer.
// runtime.noescape would do the same, but a linknamed function is not inlined,
// which costs 1-2 ns per call.
//
//go:nosplit
func noescape(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return *(*unsafe.Pointer)(unsafe.Pointer(&x))
}
