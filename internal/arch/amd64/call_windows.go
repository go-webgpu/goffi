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
	args := make([]uintptr, len(cif.ArgTypes))

	for idx := range cif.ArgTypes {
		argType := cif.ArgTypes[idx]

		switch argType.Kind {
		case types.PointerType:
			args[idx] = *(*uintptr)(avalue[idx])
		case types.SInt8Type, types.UInt8Type:
			args[idx] = uintptr(*(*uint8)(avalue[idx]))
		case types.SInt16Type, types.UInt16Type:
			args[idx] = uintptr(*(*uint16)(avalue[idx]))
		case types.SInt32Type, types.UInt32Type:
			args[idx] = uintptr(*(*uint32)(avalue[idx]))
		case types.SInt64Type, types.UInt64Type:
			args[idx] = uintptr(*(*uint64)(avalue[idx]))
		case types.FloatType:
			// Use math.Float32bits to preserve the exact 32-bit IEEE-754 pattern
			// in the XMM register slot. Widening to float64 corrupts the bit pattern
			// because callee reads only the lower 32 bits from the XMM register.
			args[idx] = uintptr(math.Float32bits(*(*float32)(avalue[idx])))
		case types.DoubleType:
			// Pass float64 as raw bit pattern
			args[idx] = *(*uintptr)(avalue[idx])
		case types.StructType:
			// Windows x64 ABI: structs of exactly 1, 2, 4, or 8 bytes are passed by
			// value (integer register / stack slot). All other sizes are passed by
			// reference — the caller passes a pointer to a copy of the struct.
			switch argType.Size {
			case 1:
				args[idx] = uintptr(*(*uint8)(avalue[idx]))
			case 2:
				args[idx] = uintptr(*(*uint16)(avalue[idx]))
			case 4:
				args[idx] = uintptr(*(*uint32)(avalue[idx]))
			case 8:
				args[idx] = *(*uintptr)(avalue[idx])
			default:
				args[idx] = uintptr(avalue[idx])
			}
		default:
			// For unknown/composite types, treat as pointer to value
			args[idx] = uintptr(avalue[idx])
		}
	}

	// Call via syscall.SyscallN — handles all args including stack args (5+).
	// On windows/amd64, its second result contains the raw XMM0 bits so callers
	// can recover scalar floating-point return values.
	ret, floatRet, _ := syscall.SyscallN(uintptr(fn), args...)

	runtime.KeepAlive(avalue)

	returnBits := ret
	if cif.ReturnType.Kind == types.FloatType || cif.ReturnType.Kind == types.DoubleType {
		returnBits = floatRet
	}

	// cerrno is always 0 on Windows: errno capture via __errno_location is not applicable.
	// Windows Win32 errors use GetLastError(); CRT errno is rarely used for Win32 APIs.
	return 0, i.handleReturn(cif, rvalue, uint64(returnBits), 0, 0, 0)
}
