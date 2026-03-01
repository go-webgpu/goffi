//go:build amd64

package amd64

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
	return classifyReturnAMD64(t, abi)
}

func (i *Implementation) ClassifyArgument(
	t *types.TypeDescriptor,
	abi types.CallingConvention,
) arch.ArgumentClassification {
	classes := classifyArgumentAMD64(t, abi)
	return arch.ArgumentClassification{
		GPRCount: classes.GPRCount,
		SSECount: classes.SSECount,
	}
}

// Return value handling (common for both Unix and Windows AMD64).
// retVal  = RAX (first integer return register)
// retVal2 = RDX (second integer return register, used for 9-16 byte struct returns)
func (i *Implementation) handleReturn(
	cif *types.CallInterface,
	rvalue unsafe.Pointer,
	retVal uint64,
	retVal2 uint64,
) error {
	if rvalue == nil || cif.ReturnType.Kind == types.VoidType {
		return nil
	}

	// Structs > 16 bytes are returned via hidden first argument (sret pointer);
	// the callee writes directly into the buffer, so nothing to do here.
	if cif.ReturnType.Kind == types.StructType && cif.ReturnType.Size > 16 {
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
		// System V AMD64 ABI struct return rules:
		//   <= 8 bytes : returned in RAX
		//   9-16 bytes : returned in RAX (low 8) + RDX (high 8)
		//   > 16 bytes : returned via hidden sret pointer (handled above)
		size := cif.ReturnType.Size
		switch {
		case size <= 8:
			*(*uint64)(rvalue) = retVal
		case size <= 16:
			// Copy RAX into first 8 bytes, RDX into remaining bytes
			*(*uint64)(rvalue) = retVal
			// Remaining bytes are in RDX; copy only what is needed
			remaining := size - 8
			src := (*[8]byte)(unsafe.Pointer(&retVal2))
			dst := (*[8]byte)(unsafe.Add(rvalue, 8))
			copy(dst[:remaining], src[:remaining])
		default:
			return types.ErrUnsupportedReturnType
		}
	default:
		return types.ErrUnsupportedReturnType
	}

	return nil
}
