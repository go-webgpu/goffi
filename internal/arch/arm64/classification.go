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
		// AAPCS64: Check HFA first - HFAs are returned in D0-D3 regardless of size.
		// Example: NSRect (4 x float64 = 32 bytes) is HFA, returned in D0-D3.
		isHFA, hfaCount := isHomogeneousFloatAggregate(t)
		if isHFA && hfaCount <= 4 {
			// Determine element type (float32 or float64)
			elemType := types.ReturnInXMM64 // default to double
			if t.Members[0].Kind == types.FloatType {
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
		// AAPCS64: Composite types
		// - HFA (Homogeneous Floating-point Aggregate): up to 4 floats/doubles in FP regs
		// - Other composites <= 16 bytes: in GP registers
		// - Larger composites: passed by reference
		//
		// IMPORTANT: Check HFA FIRST - HFAs use FP registers regardless of size.
		// Example: NSRect (4 x float64 = 32 bytes) is HFA, passed in D0-D3.
		isHFA, hfaCount := isHomogeneousFloatAggregate(t)
		if isHFA && hfaCount <= 4 {
			res.FPRCount = hfaCount
		} else if t.Size > 16 {
			// Non-HFA larger than 16 bytes: passed by reference
			res.GPRCount = 1
		} else {
			// Non-HFA up to 16 bytes: in GP registers
			res.GPRCount = int(math.Ceil(float64(t.Size) / 8))
		}
	default:
		// Integer/pointer types use GP registers (X0-X7)
		res.GPRCount = int(math.Ceil(float64(t.Size) / 8))
	}

	return res
}

// isHomogeneousFloatAggregate checks if a struct is an HFA (Homogeneous Floating-point Aggregate).
// An HFA contains 1-4 members of the same floating-point type.
func isHomogeneousFloatAggregate(t *types.TypeDescriptor) (bool, int) {
	if t.Kind != types.StructType || len(t.Members) == 0 || len(t.Members) > 4 {
		return false, 0
	}

	firstKind := t.Members[0].Kind
	if firstKind != types.FloatType && firstKind != types.DoubleType {
		return false, 0
	}

	for _, member := range t.Members {
		if member.Kind != firstKind {
			return false, 0
		}
	}

	return true, len(t.Members)
}
