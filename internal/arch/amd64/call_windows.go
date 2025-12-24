//go:build amd64 && windows

package amd64

import (
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

func (i *Implementation) Execute(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
) error {
	// Prepare registers
	gprRegs := make([]uint64, 4)  // RCX, RDX, R8, R9
	sseRegs := make([]float64, 4) // XMM0-3

	gprIndex, sseIndex := 0, 0

	// Process arguments
	for idx := range cif.ArgTypes {
		argType := cif.ArgTypes[idx]
		classification := i.ClassifyArgument(argType, cif.Convention)

		if classification.GPRCount > 0 && gprIndex < len(gprRegs) {
			// Dereference based on type - avalue[idx] points TO the value
			switch argType.Kind {
			case types.PointerType:
				gprRegs[gprIndex] = uint64(*(*uintptr)(avalue[idx]))
			case types.SInt8Type, types.UInt8Type:
				gprRegs[gprIndex] = uint64(*(*uint8)(avalue[idx]))
			case types.SInt16Type, types.UInt16Type:
				gprRegs[gprIndex] = uint64(*(*uint16)(avalue[idx]))
			case types.SInt32Type, types.UInt32Type:
				gprRegs[gprIndex] = uint64(*(*uint32)(avalue[idx]))
			case types.SInt64Type, types.UInt64Type:
				gprRegs[gprIndex] = uint64(*(*uint64)(avalue[idx]))
			default:
				// For unknown types, treat as pointer to value
				gprRegs[gprIndex] = uint64(uintptr(avalue[idx]))
			}
			gprIndex++
			continue
		}

		if classification.SSECount > 0 && sseIndex < len(sseRegs) {
			if argType.Kind == types.FloatType {
				sseRegs[sseIndex] = float64(*(*float32)(avalue[idx]))
			} else {
				sseRegs[sseIndex] = *(*float64)(avalue[idx])
			}
			sseIndex++
			continue
		}

		panic("stack arguments not implemented")
	}

	// Call via assembly
	retVal := callWin64(gprRegs, sseRegs, fn)

	// Handle return value
	return i.handleReturn(cif, rvalue, retVal)
}

func callWin64(gpr []uint64, sse []float64, fn unsafe.Pointer) uint64
