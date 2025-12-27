//go:build amd64 && windows

package amd64

import (
	"syscall"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

func (i *Implementation) Execute(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
) error {
	// Collect all arguments into a slice for syscall.SyscallN
	// Win64 ABI: first 4 in RCX/RDX/R8/R9, rest on stack
	// syscall.SyscallN handles both register and stack arguments
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
			// Pass float32 as bit pattern in 64-bit register
			f := float64(*(*float32)(avalue[idx]))
			args[idx] = *(*uintptr)(unsafe.Pointer(&f))
		case types.DoubleType:
			// Pass float64 as bit pattern
			args[idx] = *(*uintptr)(avalue[idx])
		default:
			// For unknown/composite types, treat as pointer to value
			args[idx] = uintptr(avalue[idx])
		}
	}

	// Call via syscall.SyscallN - handles all args including stack args (5+)
	ret, _, _ := syscall.SyscallN(uintptr(fn), args...)

	// Handle return value
	retVal := uint64(ret)

	// Note: Float return values (XMM0) are not captured by SyscallN.
	// This is a known limitation matching purego's behavior.

	return i.handleReturn(cif, rvalue, retVal)
}
