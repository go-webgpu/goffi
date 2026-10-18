//go:build amd64 && windows

package amd64

import (
	"math"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// Execute implements arch.FunctionCaller for Windows AMD64.
// errnoFn is always 0 on Windows (ErrnoFnAddr returns 0); cerrno is always 0.
func (i *Implementation) Execute(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
	errnoFn uintptr,
) (cerrno uintptr, err error) {
	// Win64 ABI: arguments are passed in numbered slots.
	// First 4 args: RCX, RDX, R8, R9 (integer) or XMM0-XMM3 (float).
	// Args 5+: on the stack.
	// syscall.SyscallN handles the full Win64 stack layout including shadow space.
	//
	// A struct result that is not 1, 2, 4 or 8 bytes is returned through a
	// hidden pointer to caller memory passed as the first argument, shifting
	// the others by one slot (libffi src/x86/ffiw64.c). rvalue is that memory;
	// ffi.CallFunctionContext supplies a buffer when the caller passes nil.
	sret := cif.Flags&types.ReturnViaPointer != 0
	first := 0
	if sret {
		first = 1
	}
	args := make([]uintptr, first+len(cif.ArgTypes))
	if sret {
		args[0] = uintptr(rvalue)
	}

	// Structs passed by reference get a copy: the callee owns the memory and
	// may write to it, and that must not reach the caller's Go value. The
	// copies are heap memory kept alive until the call returns.
	var copies [][]byte

	for idx := range cif.ArgTypes {
		argType := cif.ArgTypes[idx]
		slot := first + idx

		switch argType.Kind {
		case types.PointerType:
			args[slot] = *(*uintptr)(avalue[idx])
		case types.SInt8Type, types.UInt8Type:
			args[slot] = uintptr(*(*uint8)(avalue[idx]))
		case types.SInt16Type, types.UInt16Type:
			args[slot] = uintptr(*(*uint16)(avalue[idx]))
		case types.SInt32Type, types.UInt32Type, types.IntType:
			args[slot] = uintptr(*(*uint32)(avalue[idx]))
		case types.SInt64Type, types.UInt64Type:
			args[slot] = uintptr(*(*uint64)(avalue[idx]))
		case types.FloatType:
			// Use math.Float32bits to preserve the exact 32-bit IEEE-754 pattern
			// in the XMM register slot. Widening to float64 corrupts the bit pattern
			// because callee reads only the lower 32 bits from the XMM register.
			args[slot] = uintptr(math.Float32bits(*(*float32)(avalue[idx])))
		case types.DoubleType:
			// Pass float64 as raw bit pattern
			args[slot] = *(*uintptr)(avalue[idx])
		case types.StructType:
			// Windows x64 ABI: structs of exactly 1, 2, 4, or 8 bytes are passed by
			// value (integer register / stack slot). All other sizes are passed by
			// reference — the caller passes a pointer to a copy of the struct.
			switch argType.Size {
			case 1:
				args[slot] = uintptr(*(*uint8)(avalue[idx]))
			case 2:
				args[slot] = uintptr(*(*uint16)(avalue[idx]))
			case 4:
				args[slot] = uintptr(*(*uint32)(avalue[idx]))
			case 8:
				args[slot] = *(*uintptr)(avalue[idx])
			default:
				cp := make([]byte, argType.Size)
				copy(cp, unsafe.Slice((*byte)(avalue[idx]), argType.Size))
				copies = append(copies, cp)
				args[slot] = uintptr(unsafe.Pointer(unsafe.SliceData(cp)))
			}
		default:
			// For unknown/composite types, treat as pointer to value
			args[slot] = uintptr(avalue[idx])
		}
	}

	// Call via syscall.SyscallN — handles all args including stack args (5+).
	// On windows/amd64, its second result contains the raw XMM0 bits so callers
	// can recover scalar floating-point return values.
	ret, floatRet, _ := syscall.SyscallN(uintptr(fn), args...)

	runtime.KeepAlive(avalue)
	runtime.KeepAlive(rvalue)
	runtime.KeepAlive(copies)

	returnBits := ret
	if cif.ReturnType.Kind == types.FloatType || cif.ReturnType.Kind == types.DoubleType {
		returnBits = floatRet
	}

	// cerrno is always 0 on Windows: errno capture via __errno_location is not applicable.
	// Windows Win32 errors use GetLastError(); CRT errno is rarely used for Win32 APIs.
	return 0, i.handleReturn(cif, rvalue, uint64(returnBits), 0, 0, 0)
}
