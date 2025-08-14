//go:build amd64

package amd64

import (
	"math"

	"github.com/go-webgpu/goffi/types"
)

type classification struct {
	GPRCount int
	SSECount int
}

// classifyReturnAMD64 for x86_64
func classifyReturnAMD64(t *types.TypeDescriptor, abi types.CallingConvention) int {
	switch t.Kind {
	case types.VoidType:
		return types.ReturnVoid
	case types.FloatType:
		return types.ReturnInXMM32
	case types.DoubleType:
		return types.ReturnInXMM64
	case types.StructType:
		switch t.Size {
		case 1:
			return types.ReturnSInt8
		case 2:
			return types.ReturnSInt16
		case 4:
			return types.ReturnSInt32
		case 8:
			return types.ReturnInt64
		default:
			return types.ReturnViaPointer | types.ReturnVoid
		}
	default:
		if t.Size <= 8 {
			return types.ReturnInt64
		}
		return types.ReturnViaPointer | types.ReturnVoid
	}
}

// classifyArgumentAMD64 for x86_64
func classifyArgumentAMD64(t *types.TypeDescriptor, abi types.CallingConvention) classification {
	res := classification{}
	switch t.Kind {
	case types.FloatType, types.DoubleType:
		res.SSECount = 1
	case types.StructType:
		if t.Size > 16 {
			res.GPRCount = int(math.Ceil(float64(t.Size) / 8))
		} else {
			res.GPRCount = int(math.Ceil(float64(t.Size) / 8))
			for _, el := range t.Members {
				if el.Kind == types.FloatType || el.Kind == types.DoubleType {
					res.SSECount += 1
					res.GPRCount -= 1
					break
				}
			}
		}
	default:
		res.GPRCount = int(math.Ceil(float64(t.Size) / 8))
	}
	return res
}
