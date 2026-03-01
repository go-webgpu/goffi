//go:build amd64 && windows

package amd64

import (
	"math"
	"runtime"
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
		default:
			// For unknown/composite types, treat as pointer to value
			args[idx] = uintptr(avalue[idx])
		}
	}

	// Call via syscall.SyscallN — handles all args including stack args (5+).
	ret, _, _ := syscall.SyscallN(uintptr(fn), args...)

	runtime.KeepAlive(avalue)

	// Handle return value.
	// Note: float return values in XMM0 are not captured by syscall.SyscallN on Windows.
	// This is a known limitation: Go's syscall package on Windows only exposes RAX (ret).
	// Float-returning C functions on Windows require a custom assembly wrapper to capture
	// XMM0. Since this requires significant additional infrastructure and matches purego's
	// documented limitation, it is recorded as a known limitation for v0.4.1.
	// See: TASK-019, GAP-7. Workaround: use integer return type and reinterpret bits.
	return i.handleReturn(cif, rvalue, uint64(ret), 0)
}
