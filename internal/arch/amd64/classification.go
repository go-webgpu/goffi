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
		// System V: a struct of one eightbyte whose members are all float or
		// double is SSE class and comes back in XMM0 ({float}, {double},
		// {float, float}). Win64 returns 1/2/4/8-byte structs in RAX whatever
		// their members.
		if runtime.GOOS != "windows" && t.Size <= 8 && classifyEightbyte(t, 0, t.Size) {
			return types.ReturnInXMM64
		}
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

// classifyEightbyte reports whether the eightbyte [startOff, endOff) of struct
// t is SSE class: every scalar member that starts in it, at any nesting depth,
// is a float or double (System V AMD64 ABI §3.2.3: the classes of all fields
// in an eightbyte are merged and INTEGER wins; libffi classify_argument walks
// nested structs the same way). An eightbyte with no member is not SSE.
func classifyEightbyte(t *types.TypeDescriptor, startOff, endOff uintptr) bool {
	hasField, allFloat := false, true
	walkScalarMembers(t, 0, func(off uintptr, m *types.TypeDescriptor) bool {
		if off < startOff || off >= endOff {
			return true
		}
		hasField = true
		if m.Kind != types.FloatType && m.Kind != types.DoubleType {
			allFloat = false
			return false
		}
		return true
	})
	return hasField && allFloat
}

// walkScalarMembers calls fn with the offset (from base) of every non-struct
// member of struct t, nested structs included, in layout order, until fn
// returns false. Offsets follow each member's alignment, as
// PrepareCallInterface lays the struct out. It returns false when fn stopped
// the walk.
func walkScalarMembers(t *types.TypeDescriptor, base uintptr, fn func(off uintptr, m *types.TypeDescriptor) bool) bool {
	var offset uintptr
	for _, m := range t.Members {
		if m == nil {
			continue
		}
		if m.Alignment > 0 {
			offset = (offset + m.Alignment - 1) &^ (m.Alignment - 1)
		}
		if m.Kind == types.StructType {
			if !walkScalarMembers(m, base+offset, fn) {
				return false
			}
		} else if !fn(base+offset, m) {
			return false
		}
		offset += m.Size
	}
	return true
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
