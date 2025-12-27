//go:build amd64 && windows

package amd64

import (
	"unsafe"

	"github.com/go-webgpu/goffi/internal/syscall"
	"github.com/go-webgpu/goffi/types"
)

func (i *Implementation) Execute(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
) error {
	// Prepare registers - Win64 ABI: 4 GPR (RCX, RDX, R8, R9), 4 SSE (XMM0-3)
	var gpr [4]uintptr
	var sse [4]float64

	gprIndex, sseIndex := 0, 0

	// Process arguments
	for idx := range cif.ArgTypes {
		argType := cif.ArgTypes[idx]
		classification := i.ClassifyArgument(argType, cif.Convention)

		if classification.GPRCount > 0 && gprIndex < len(gpr) {
			// Dereference based on type - avalue[idx] points TO the value
			switch argType.Kind {
			case types.PointerType:
				gpr[gprIndex] = *(*uintptr)(avalue[idx])
			case types.SInt8Type, types.UInt8Type:
				gpr[gprIndex] = uintptr(*(*uint8)(avalue[idx]))
			case types.SInt16Type, types.UInt16Type:
				gpr[gprIndex] = uintptr(*(*uint16)(avalue[idx]))
			case types.SInt32Type, types.UInt32Type:
				gpr[gprIndex] = uintptr(*(*uint32)(avalue[idx]))
			case types.SInt64Type, types.UInt64Type:
				gpr[gprIndex] = uintptr(*(*uint64)(avalue[idx]))
			default:
				// For unknown types, treat as pointer to value
				gpr[gprIndex] = uintptr(avalue[idx])
			}
			gprIndex++
			continue
		}

		if classification.SSECount > 0 && sseIndex < len(sse) {
			if argType.Kind == types.FloatType {
				sse[sseIndex] = float64(*(*float32)(avalue[idx]))
			} else {
				sse[sseIndex] = *(*float64)(avalue[idx])
			}
			sseIndex++
			continue
		}

		panic("stack arguments not implemented")
	}

	// Call via syscall with proper cgocall stack handling
	ret, fret := syscall.CallWin64(uintptr(fn), gpr, sse)

	// Handle return value based on type
	retVal := uint64(ret)

	// For float returns, use the float value
	if cif.ReturnType.Kind == types.FloatType || cif.ReturnType.Kind == types.DoubleType {
		retVal = *(*uint64)(unsafe.Pointer(&fret))
	}

	return i.handleReturn(cif, rvalue, retVal)
}
