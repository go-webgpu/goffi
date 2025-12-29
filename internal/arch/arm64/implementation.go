//go:build arm64

package arm64

import (
	"unsafe"

	"github.com/go-webgpu/goffi/internal/arch"
	"github.com/go-webgpu/goffi/types"
)

type Implementation struct{}

func init() {
	arch.Register(&Implementation{}, &Implementation{})
}

func (i *Implementation) align(value, alignment uintptr) uintptr {
	return (value + alignment - 1) &^ (alignment - 1)
}

func (i *Implementation) ClassifyReturn(
	t *types.TypeDescriptor,
	abi types.CallingConvention,
) int {
	return classifyReturnARM64(t, abi)
}

func (i *Implementation) ClassifyArgument(
	t *types.TypeDescriptor,
	abi types.CallingConvention,
) arch.ArgumentClassification {
	classes := classifyArgumentARM64(t, abi)
	return arch.ArgumentClassification{
		GPRCount: classes.GPRCount,
		SSECount: classes.FPRCount, // ARM64 uses FPR, but we map to SSECount for interface compatibility
	}
}

// Return value handling for ARM64 (AAPCS64)
// fret contains D0-D3 float register values (for HFA returns).
func (i *Implementation) handleReturn(
	cif *types.CallInterface,
	rvalue unsafe.Pointer,
	retVal uint64,
	fret [4]float64,
) error {
	if rvalue == nil || cif.ReturnType.Kind == types.VoidType {
		return nil
	}

	// Handle sret (large non-HFA struct return via X8)
	// Callee already wrote directly to rvalue buffer via X8 pointer.
	// Nothing to do here - data is already in place.
	if cif.Flags&types.ReturnViaPointer != 0 {
		return nil
	}

	// Handle HFA returns (1-4 floats/doubles in D0-D3)
	if cif.Flags&(types.ReturnHFA2|types.ReturnHFA3|types.ReturnHFA4) != 0 {
		return i.handleHFAReturn(cif, rvalue, fret)
	}

	switch cif.ReturnType.Kind {
	case types.FloatType:
		// Single float in D0
		*(*float32)(rvalue) = float32(fret[0])
	case types.DoubleType:
		// Single double in D0
		*(*float64)(rvalue) = fret[0]
	case types.UInt8Type:
		*(*uint8)(rvalue) = uint8(retVal)
	case types.SInt8Type:
		*(*int8)(rvalue) = int8(retVal)
	case types.UInt16Type:
		*(*uint16)(rvalue) = uint16(retVal)
	case types.SInt16Type:
		*(*int16)(rvalue) = int16(retVal)
	case types.UInt32Type:
		*(*uint32)(rvalue) = uint32(retVal)
	case types.SInt32Type:
		*(*int32)(rvalue) = int32(retVal)
	case types.UInt64Type, types.SInt64Type, types.PointerType:
		*(*uint64)(rvalue) = retVal
	case types.StructType:
		if cif.ReturnType.Size <= 8 {
			*(*uint64)(rvalue) = retVal
		} else if cif.ReturnType.Size <= 16 {
			// 9-16 byte struct returned in X0-X1
			dest := (*[2]uint64)(rvalue)
			dest[0] = retVal
			// X1 is not returned by our current syscall, so this is partial
			// TODO: Support X1 return value if needed
		} else {
			return types.ErrUnsupportedReturnType
		}
	default:
		return types.ErrUnsupportedReturnType
	}

	return nil
}

// handleHFAReturn handles HFA (Homogeneous Floating-point Aggregate) returns.
// HFA structs with 2-4 floats/doubles are returned in D0-D3.
func (i *Implementation) handleHFAReturn(
	cif *types.CallInterface,
	rvalue unsafe.Pointer,
	fret [4]float64,
) error {
	// Determine HFA count from flags
	var hfaCount int
	switch {
	case cif.Flags&types.ReturnHFA4 != 0:
		hfaCount = 4
	case cif.Flags&types.ReturnHFA3 != 0:
		hfaCount = 3
	case cif.Flags&types.ReturnHFA2 != 0:
		hfaCount = 2
	default:
		hfaCount = 1
	}

	// Determine element type (float32 or float64)
	isFloat32 := cif.Flags&types.ReturnInXMM32 != 0

	if isFloat32 {
		// HFA with float32 elements
		dest := (*[4]float32)(rvalue)
		for idx := 0; idx < hfaCount; idx++ {
			dest[idx] = float32(fret[idx])
		}
	} else {
		// HFA with float64 elements (e.g., NSRect = 4 x float64)
		dest := (*[4]float64)(rvalue)
		for idx := 0; idx < hfaCount; idx++ {
			dest[idx] = fret[idx]
		}
	}

	return nil
}
