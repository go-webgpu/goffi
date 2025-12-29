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

	// Determine if we need to pass r8 for large struct return (sret)
	var r8 uintptr
	if cif.Flags&types.ReturnViaPointer != 0 && rvalue != nil {
		// For sret, pass rvalue pointer in X8 - callee writes directly to it
		r8 = uintptr(rvalue)
	}

	// Call via our ARM64 syscall wrapper
	ret, fret := gosyscall.Call8Float(uintptr(fn), gpr, fpr, r8)

	// Handle return value based on type
	return i.handleReturn(cif, rvalue, uint64(ret), fret)
}
