//go:build arm64

package arm64

import (
	"math"

	"github.com/go-webgpu/goffi/types"
)

type classification struct {
	GPRCount int // X0-X7 (8 registers)
	FPRCount int // D0-D7/V0-V7 (8 registers)
}

type regClass uint8

const (
	classNone  regClass = 0
	classFloat regClass = 1
	classInt   regClass = 2
)

func alignOffset(value, alignment uintptr) uintptr {
	if alignment == 0 {
		return value
	}
	return (value + alignment - 1) &^ (alignment - 1)
}

func ensureStructLayout(desc *types.TypeDescriptor) (size, align uintptr) {
	if desc == nil {
		return 0, 1
	}
	if desc.Kind != types.StructType {
		if desc.Alignment == 0 {
			if desc.Size != 0 {
				desc.Alignment = desc.Size
			} else {
				desc.Alignment = 1
			}
		}
		return desc.Size, desc.Alignment
	}

	if desc.Size != 0 && desc.Alignment != 0 {
		return desc.Size, desc.Alignment
	}

	var (
		offset   uintptr
		maxAlign uintptr = 1
	)
	for _, member := range desc.Members {
		if member == nil {
			continue
		}
		mSize, mAlign := ensureStructLayout(member)
		if mAlign == 0 {
			mAlign = 1
		}
		offset = alignOffset(offset, mAlign)
		offset += mSize
		if mAlign > maxAlign {
			maxAlign = mAlign
		}
	}

	size = alignOffset(offset, maxAlign)
	if desc.Size == 0 {
		desc.Size = size
	}
	if desc.Alignment == 0 {
		desc.Alignment = maxAlign
	}
	return desc.Size, desc.Alignment
}

// classifyReturnARM64 determines how a return value is passed according to AAPCS64.
// Return values:
//   - X0-X1: Integer/pointer returns (up to 16 bytes)
//   - D0-D3: Floating-point returns (including HFA with 1-4 floats/doubles)
//   - X8: Indirect result location (for larger non-HFA returns)
func classifyReturnARM64(t *types.TypeDescriptor, abi types.CallingConvention) int {
	switch t.Kind {
	case types.VoidType:
		return types.ReturnVoid
	case types.FloatType:
		return types.ReturnInXMM32 // Uses D0 on ARM64
	case types.DoubleType:
		return types.ReturnInXMM64 // Uses D0 on ARM64
	case types.StructType:
		ensureStructLayout(t)
		// AAPCS64: Check HFA first - HFAs are returned in D0-D3 regardless of size.
		// Example: NSRect (4 x float64 = 32 bytes) is HFA, returned in D0-D3.
		isHFA, hfaCount, elemKind := isHomogeneousFloatAggregate(t)
		if isHFA && hfaCount <= 4 {
			// Determine element type (float32 or float64)
			elemType := types.ReturnInXMM64 // default to double
			if elemKind == types.FloatType {
				elemType = types.ReturnInXMM32
			}

			switch hfaCount {
			case 1:
				return elemType // Single element in D0
			case 2:
				return types.ReturnHFA2 | elemType
			case 3:
				return types.ReturnHFA3 | elemType
			case 4:
				return types.ReturnHFA4 | elemType
			}
		}

		// Non-HFA composites: <= 16 bytes in X0-X1, larger via X8 (indirect)
		switch {
		case t.Size <= 8:
			return types.ReturnInt64
		case t.Size <= 16:
			return types.ReturnInt64 // Returned in X0-X1
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

// classifyArgumentARM64 determines how an argument is passed according to AAPCS64.
// Arguments are passed in:
//   - X0-X7: First 8 integer/pointer arguments
//   - D0-D7: First 8 floating-point arguments
//   - Stack: Additional arguments (16-byte aligned)
func classifyArgumentARM64(t *types.TypeDescriptor, abi types.CallingConvention) classification {
	res := classification{}

	switch t.Kind {
	case types.FloatType, types.DoubleType:
		// Floating-point arguments use FP registers (D0-D7)
		res.FPRCount = 1
	case types.StructType:
		ensureStructLayout(t)
		// AAPCS64: Composite types
		// - HFA (Homogeneous Floating-point Aggregate): up to 4 floats/doubles in FP regs
		// - Other composites <= 16 bytes: in GP registers
		// - Larger composites: passed by reference
		//
		// IMPORTANT: Check HFA FIRST - HFAs use FP registers regardless of size.
		// Example: NSRect (4 x float64 = 32 bytes) is HFA, passed in D0-D3.
		isHFA, hfaCount, _ := isHomogeneousFloatAggregate(t)
		if isHFA && hfaCount <= 4 {
			res.FPRCount = hfaCount
		} else if t.Size > 16 {
			// Non-HFA larger than 16 bytes: passed by reference
			res.GPRCount = 1
		} else {
			// Non-HFA up to 16 bytes: mixed int/float register usage
			res.GPRCount, res.FPRCount = countStructRegUsage(t)
		}
	default:
		// Integer/pointer types use GP registers (X0-X7)
		res.GPRCount = int(math.Ceil(float64(t.Size) / 8))
	}

	return res
}

func countStructRegUsage(desc *types.TypeDescriptor) (intCount, floatCount int) {
	if desc == nil || desc.Kind != types.StructType {
		return 0, 0
	}
	ensureStructLayout(desc)

	var (
		shift uint
		class regClass
	)

	flush := func() {
		if class == classNone {
			shift = 0
			return
		}
		if class == classFloat {
			floatCount++
		} else {
			intCount++
		}
		shift = 0
		class = classNone
	}

	var walk func(cur *types.TypeDescriptor)
	walk = func(cur *types.TypeDescriptor) {
		if cur == nil {
			return
		}
		if cur.Kind == types.StructType {
			offset := uintptr(0)
			for _, member := range cur.Members {
				if member == nil {
					continue
				}
				offset = alignOffset(offset, member.Alignment)
				walk(member)
				offset += member.Size
			}
			return
		}

		alignBits := uint(cur.Alignment*8 - 1)
		shift = (shift + alignBits) &^ alignBits
		if shift >= 64 {
			flush()
			shift = 0
		}

		switch cur.Kind {
		case types.FloatType:
			if class == classFloat {
				flush()
			}
			shift += 32
			class |= classFloat
		case types.DoubleType:
			flush()
			floatCount++
			shift = 0
			class = classNone
		case types.UInt8Type, types.SInt8Type:
			shift += 8
			class |= classInt
		case types.UInt16Type, types.SInt16Type:
			shift += 16
			class |= classInt
		case types.UInt32Type, types.SInt32Type:
			shift += 32
			class |= classInt
		case types.UInt64Type, types.SInt64Type, types.PointerType:
			flush()
			intCount++
			shift = 0
			class = classNone
		default:
			// Unsupported kinds are treated as int-sized to avoid undercounting.
			flush()
			intCount++
			shift = 0
			class = classNone
		}
	}

	walk(desc)
	if class != classNone {
		flush()
	}
	return intCount, floatCount
}

// isHomogeneousFloatAggregate checks if a struct is an HFA (Homogeneous Floating-point Aggregate).
// An HFA contains 1-4 total floating-point members (float32 or float64) of the same type, possibly nested.
func isHomogeneousFloatAggregate(t *types.TypeDescriptor) (bool, int, types.TypeKind) {
	if t.Kind != types.StructType {
		return false, 0, types.VoidType
	}

	const invalidKind types.TypeKind = -1

	var (
		elementKind types.TypeKind = invalidKind
		totalCount  int
	)

	var walk func(desc *types.TypeDescriptor) bool
	walk = func(desc *types.TypeDescriptor) bool {
		switch desc.Kind {
		case types.FloatType, types.DoubleType:
			if elementKind == invalidKind {
				elementKind = desc.Kind
			} else if desc.Kind != elementKind {
				return false
			}
			totalCount++
			return totalCount <= 4
		case types.StructType:
			if len(desc.Members) == 0 {
				return false
			}
			for _, member := range desc.Members {
				if !walk(member) {
					return false
				}
			}
			return true
		default:
			return false
		}
	}

	if !walk(t) || elementKind == invalidKind || totalCount == 0 {
		return false, 0, types.VoidType
	}

	return true, totalCount, elementKind
}
