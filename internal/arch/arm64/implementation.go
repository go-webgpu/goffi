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
func (i *Implementation) handleReturn(
	cif *types.CallInterface,
	rvalue unsafe.Pointer,
	retVal uint64,
) error {
	if rvalue == nil || cif.ReturnType.Kind == types.VoidType {
		return nil
	}

	if cif.Flags&types.ReturnViaPointer != 0 {
		*(*unsafe.Pointer)(rvalue) = unsafe.Pointer(uintptr(retVal))
		return nil
	}

	switch cif.ReturnType.Kind {
	case types.FloatType:
		*(*float32)(rvalue) = *(*float32)(unsafe.Pointer(&retVal))
	case types.DoubleType:
		*(*float64)(rvalue) = *(*float64)(unsafe.Pointer(&retVal))
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
		} else {
			return types.ErrUnsupportedReturnType
		}
	default:
		return types.ErrUnsupportedReturnType
	}

	return nil
}
