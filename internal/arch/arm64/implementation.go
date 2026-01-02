//go:build arm64

package arm64

import (
	"math"
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
// fret contains raw D0-D3 bit patterns (for float and HFA returns).
func (i *Implementation) handleReturn(
	cif *types.CallInterface,
	rvalue unsafe.Pointer,
	retLo uint64,
	retHi uint64,
	fret [4]uint64,
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
		// Single float in S0 (low 32 bits of D0)
		*(*float32)(rvalue) = math.Float32frombits(uint32(fret[0]))
	case types.DoubleType:
		// Single double in D0
		*(*float64)(rvalue) = math.Float64frombits(fret[0])
	case types.UInt8Type:
		*(*uint8)(rvalue) = uint8(retLo)
	case types.SInt8Type:
		*(*int8)(rvalue) = int8(retLo)
	case types.UInt16Type:
		*(*uint16)(rvalue) = uint16(retLo)
	case types.SInt16Type:
		*(*int16)(rvalue) = int16(retLo)
	case types.UInt32Type:
		*(*uint32)(rvalue) = uint32(retLo)
	case types.SInt32Type:
		*(*int32)(rvalue) = int32(retLo)
	case types.UInt64Type, types.SInt64Type, types.PointerType:
		*(*uint64)(rvalue) = retLo
	case types.StructType:
		if cif.ReturnType.Size <= 8 {
			*(*uint64)(rvalue) = retLo
		} else if cif.ReturnType.Size <= 16 {
			// 9-16 byte struct returned in X0-X1
			dest := (*[2]uint64)(rvalue)
			dest[0] = retLo
			dest[1] = retHi
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
	fret [4]uint64,
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

	// Determine element type (float32 or float64) from the return descriptor.
	// Return flags overlap (ReturnInXMM64 includes ReturnInXMM32 bit), so rely on HFA metadata.
	elemKind := types.DoubleType
	if cif.ReturnType != nil && cif.ReturnType.Kind == types.StructType {
		if isHFA, _, kind := isHomogeneousFloatAggregate(cif.ReturnType); isHFA {
			elemKind = kind
		}
	}
	isFloat32 := elemKind == types.FloatType

	if isFloat32 {
		dest := (*[4]float32)(rvalue)
		for idx := 0; idx < hfaCount; idx++ {
			dest[idx] = math.Float32frombits(uint32(fret[idx]))
		}
		return nil
	}

	dest := (*[4]float64)(rvalue)
	for idx := 0; idx < hfaCount; idx++ {
		dest[idx] = math.Float64frombits(fret[idx])
	}
	return nil
}
