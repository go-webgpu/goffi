//go:build amd64

package amd64

import (
	"math"
	"runtime"

	"github.com/go-webgpu/goffi/types"
)

type classification struct {
	GPRCount int
	SSECount int
}

// classifyReturnAMD64 for x86_64.
// Implements SysV AMD64 ABI §3.2.3 return-value classification.
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
			if t.Size > 16 || runtime.GOOS == "windows" {
				// MEMORY class (>16B) or Windows (all structs >8B): sret pointer.
				// Win64 ABI: structs not exactly 1/2/4/8 bytes are returned by reference.
				return types.ReturnViaPointer | types.ReturnVoid
			}
			// 9-16B on Unix: classify each eightbyte independently per SysV ABI §3.2.3.
			// INTEGER wins over SSE within an eightbyte.
			eb0SSE := classifyEightbyte(t, 0, 8)
			eb1SSE := classifyEightbyte(t, 8, t.Size)
			switch {
			case eb0SSE && eb1SSE:
				return types.ReturnStXmm0Xmm1 // {SSE, SSE}     — XMM0 : XMM1
			case eb0SSE:
				return types.ReturnStXmm0Rax // {SSE, INTEGER} — XMM0 : RAX
			case eb1SSE:
				return types.ReturnStRaxXmm0 // {INTEGER, SSE} — RAX  : XMM0
			default:
				return types.ReturnStRaxRdx // {INTEGER, INTEGER} — RAX : RDX
			}
		}
	default:
		if t.Size <= 8 {
			return types.ReturnInt64
		}
		return types.ReturnViaPointer | types.ReturnVoid
	}
}

// isStructAllFloats returns true if every member of a flat struct is float or double.
// Per System V AMD64 ABI §3.2.3: if any member in an eightbyte is INTEGER class,
// the entire eightbyte is classified as INTEGER (INTEGER wins over SSE).
func isStructAllFloats(t *types.TypeDescriptor) bool {
	if len(t.Members) == 0 {
		return false
	}
	for _, m := range t.Members {
		if m.Kind != types.FloatType && m.Kind != types.DoubleType {
			return false
		}
	}
	return true
}

// classifyEightbyte returns true if all struct fields whose offset falls within
// [startOff, endOff) are SSE types (float or double).
// Returns false if any field in the range is INTEGER class, or if no fields lie in the range.
func classifyEightbyte(t *types.TypeDescriptor, startOff, endOff uintptr) bool {
	var offset uintptr
	allFloat := true
	hasField := false
	for _, m := range t.Members {
		if m == nil {
			continue
		}
		if m.Alignment > 0 {
			offset = (offset + m.Alignment - 1) &^ (m.Alignment - 1)
		}
		if offset >= startOff && offset < endOff {
			hasField = true
			if m.Kind != types.FloatType && m.Kind != types.DoubleType {
				allFloat = false
				break
			}
		}
		offset += m.Size
	}
	return hasField && allFloat
}

// classifyArgumentAMD64 for x86_64
func classifyArgumentAMD64(t *types.TypeDescriptor, abi types.CallingConvention) classification {
	res := classification{}
	switch t.Kind {
	case types.FloatType, types.DoubleType:
		res.SSECount = 1
	case types.StructType:
		if t.Size > 16 {
			// MEMORY class: passed on the stack. No GP or SSE registers consumed.
			// The caller copies the struct bytes; the callee receives a copy on its stack frame.
		} else {
			// Walk members to classify each eightbyte independently.
			// System V AMD64 ABI §3.2.3 merge rule: INTEGER wins over SSE.
			nEightbytes := int(math.Ceil(float64(t.Size) / 8))
			for eb := 0; eb < nEightbytes; eb++ {
				startOff := uintptr(eb * 8)
				endOff := startOff + 8
				if endOff > t.Size {
					endOff = t.Size
				}
				if classifyEightbyte(t, startOff, endOff) {
					res.SSECount++
				} else {
					res.GPRCount++
				}
			}
		}
	default:
		res.GPRCount = int(math.Ceil(float64(t.Size) / 8))
	}
	return res
}
