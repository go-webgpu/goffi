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
// fret    = XMM0 float return value (for float/double types and SSE eightbytes)
// fret2   = XMM1 second float return value (for {SSE, SSE} 9-16B struct returns, e.g. NSPoint)
func (i *Implementation) handleReturn(
	cif *types.CallInterface,
	rvalue unsafe.Pointer,
	retVal uint64,
	retVal2 uint64,
	fret float64,
	fret2 float64,
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
		// Double-indirection to satisfy checkptr (go.dev/issue/58625).
		*(*unsafe.Pointer)(rvalue) = *(*unsafe.Pointer)(unsafe.Pointer(&retVal))
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
		//   <= 8 bytes : returned in RAX (any eightbyte class, since there is only one)
		//   9-16 bytes : two eightbytes, each classified as INTEGER or SSE independently.
		//                The return flag encodes which register pair was used:
		//                  ReturnStRaxRdx   → {INTEGER, INTEGER} — RAX  : RDX
		//                  ReturnStRaxXmm0  → {INTEGER, SSE}     — RAX  : XMM0
		//                  ReturnStXmm0Rax  → {SSE, INTEGER}     — XMM0 : RAX
		//                  ReturnStXmm0Xmm1 → {SSE, SSE}         — XMM0 : XMM1  (e.g. NSPoint)
		//   > 16 bytes : returned via hidden sret pointer (handled above before the switch)
		size := cif.ReturnType.Size
		if size <= 8 {
			*(*uint64)(rvalue) = retVal
			break
		}
		// 9-16B: reconstruct from the correct register pair.
		remaining := size - 8
		switch cif.Flags {
		case types.ReturnStXmm0Xmm1:
			// eightbyte0 from XMM0, eightbyte1 from XMM1
			fretBits := *(*uint64)(unsafe.Pointer(&fret))
			*(*uint64)(rvalue) = fretBits
			fret2Bits := *(*uint64)(unsafe.Pointer(&fret2))
			copy((*[8]byte)(unsafe.Add(rvalue, 8))[:remaining], (*[8]byte)(unsafe.Pointer(&fret2Bits))[:remaining])
		case types.ReturnStXmm0Rax:
			// eightbyte0 from XMM0, eightbyte1 from RAX
			fretBits := *(*uint64)(unsafe.Pointer(&fret))
			*(*uint64)(rvalue) = fretBits
			copy((*[8]byte)(unsafe.Add(rvalue, 8))[:remaining], (*[8]byte)(unsafe.Pointer(&retVal))[:remaining])
		case types.ReturnStRaxXmm0:
			// eightbyte0 from RAX, eightbyte1 from XMM0
			*(*uint64)(rvalue) = retVal
			fretBits := *(*uint64)(unsafe.Pointer(&fret))
			copy((*[8]byte)(unsafe.Add(rvalue, 8))[:remaining], (*[8]byte)(unsafe.Pointer(&fretBits))[:remaining])
		default:
			// ReturnStRaxRdx (and legacy/unset Flags): {INTEGER, INTEGER} — RAX : RDX
			*(*uint64)(rvalue) = retVal
			copy((*[8]byte)(unsafe.Add(rvalue, 8))[:remaining], (*[8]byte)(unsafe.Pointer(&retVal2))[:remaining])
		}
	default:
		return types.ErrUnsupportedReturnType
	}

	return nil
}
