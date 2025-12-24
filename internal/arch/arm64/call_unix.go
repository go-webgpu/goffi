//go:build arm64 && (linux || darwin)

// Unix implementation using AAPCS64 ABI (Linux, macOS on ARM64)
// This implementation follows the ARM64 Procedure Call Standard.

package arm64

import (
	"unsafe"

	gosyscall "github.com/go-webgpu/goffi/internal/syscall"
	"github.com/go-webgpu/goffi/types"
)

func (i *Implementation) Execute(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
) error {
	// Prepare register arguments following AAPCS64
	// X0-X7: 8 integer/pointer registers
	// D0-D7: 8 floating-point registers
	var gpr [8]uintptr
	var fpr [8]float64

	gprIdx := 0
	fprIdx := 0

	// Map arguments to registers
	for idx, argType := range cif.ArgTypes {
		if idx >= len(avalue) {
			break
		}

		switch argType.Kind {
		case types.FloatType:
			if fprIdx < 8 {
				fpr[fprIdx] = float64(*(*float32)(avalue[idx]))
				fprIdx++
			}
		case types.DoubleType:
			if fprIdx < 8 {
				fpr[fprIdx] = *(*float64)(avalue[idx])
				fprIdx++
			}
		case types.PointerType:
			if gprIdx < 8 {
				gpr[gprIdx] = *(*uintptr)(avalue[idx])
				gprIdx++
			}
		case types.SInt8Type, types.UInt8Type:
			if gprIdx < 8 {
				gpr[gprIdx] = uintptr(*(*uint8)(avalue[idx]))
				gprIdx++
			}
		case types.SInt16Type, types.UInt16Type:
			if gprIdx < 8 {
				gpr[gprIdx] = uintptr(*(*uint16)(avalue[idx]))
				gprIdx++
			}
		case types.SInt32Type, types.UInt32Type:
			if gprIdx < 8 {
				gpr[gprIdx] = uintptr(*(*uint32)(avalue[idx]))
				gprIdx++
			}
		case types.SInt64Type, types.UInt64Type:
			if gprIdx < 8 {
				gpr[gprIdx] = uintptr(*(*uint64)(avalue[idx]))
				gprIdx++
			}
		default:
			// For unknown types, pass as pointer
			if gprIdx < 8 {
				gpr[gprIdx] = uintptr(avalue[idx])
				gprIdx++
			}
		}
	}

	// Call via our ARM64 syscall wrapper
	ret, fret := gosyscall.Call8Float(uintptr(fn), gpr, fpr)

	// Handle return value based on type
	retVal := uint64(ret)

	// For float returns, use the float value
	if cif.ReturnType.Kind == types.FloatType || cif.ReturnType.Kind == types.DoubleType {
		retVal = *(*uint64)(unsafe.Pointer(&fret))
	}

	return i.handleReturn(cif, rvalue, retVal)
}
