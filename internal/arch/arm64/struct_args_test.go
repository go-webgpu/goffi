//go:build arm64

package arm64

import (
	"testing"

	"github.com/go-webgpu/goffi/types"
)

// Register and stack placement of struct arguments is tested in layout_test.go
// (layoutArgs) and abi_capture_test.go (Execute). These tests cover the
// per-argument classification PrepareCallInterface uses.

func TestClassifyArgumentMixedSmallStructUsesXRegisters(t *testing.T) {
	// {int64, double} is not an HFA: AAPCS64 passes it as raw bytes in two
	// X registers; the double does not go to a V register.
	desc := &types.TypeDescriptor{
		Kind: types.StructType,
		Members: []*types.TypeDescriptor{
			types.SInt64TypeDescriptor,
			types.DoubleTypeDescriptor,
		},
	}

	class := classifyArgumentARM64(desc, types.UnixCallingConvention)
	if class.GPRCount != 2 || class.FPRCount != 0 {
		t.Fatalf("classifyArgumentARM64 {int64, double} = (GPR=%d,FPR=%d), want (2,0)",
			class.GPRCount, class.FPRCount)
	}
}

func TestClassifyArgumentZeroAlignMembers(t *testing.T) {
	u64 := *types.UInt64TypeDescriptor
	u64.Alignment = 0

	desc := &types.TypeDescriptor{
		Kind: types.StructType,
		Members: []*types.TypeDescriptor{
			&u64,
			&u64,
		},
	}

	class := classifyArgumentARM64(desc, types.UnixCallingConvention)
	if class.GPRCount != 2 || class.FPRCount != 0 {
		t.Fatalf("classifyArgumentARM64 zero-align = (GPR=%d,FPR=%d), want (2,0)",
			class.GPRCount, class.FPRCount)
	}
	if desc.Alignment != 8 {
		t.Fatalf("computed alignment = %d, want 8", desc.Alignment)
	}
	if desc.Size != 16 {
		t.Fatalf("computed size = %d, want 16", desc.Size)
	}
}

func TestClassifyArgumentLargeStructSizeUnknown(t *testing.T) {
	desc := &types.TypeDescriptor{
		Kind: types.StructType,
		// Size intentionally left as 0 to verify classification falls back
		// to by-reference for large non-HFA structs.
		Members: []*types.TypeDescriptor{
			types.UInt64TypeDescriptor,
			types.UInt64TypeDescriptor,
			types.UInt64TypeDescriptor, // 24 bytes total
		},
	}

	class := classifyArgumentARM64(desc, types.UnixCallingConvention)
	if class.GPRCount != 1 || class.FPRCount != 0 {
		t.Fatalf("classifyArgumentARM64 size=0 large struct = (GPR=%d,FPR=%d), want (1,0)",
			class.GPRCount, class.FPRCount)
	}
}

func TestClassifyArgumentLargeStructNonHFAByRef(t *testing.T) {
	desc := &types.TypeDescriptor{
		Kind: types.StructType,
		// Mixed floats/ints (non-HFA), size intentionally omitted.
		Members: []*types.TypeDescriptor{
			types.DoubleTypeDescriptor,
			types.UInt32TypeDescriptor,
			types.UInt32TypeDescriptor,
			types.UInt32TypeDescriptor, // 24 bytes total after alignment
		},
	}

	class := classifyArgumentARM64(desc, types.UnixCallingConvention)
	if class.GPRCount != 1 || class.FPRCount != 0 {
		t.Fatalf("classifyArgumentARM64 large non-HFA = (GPR=%d,FPR=%d), want (1,0)",
			class.GPRCount, class.FPRCount)
	}
	if desc.Size != 24 {
		t.Fatalf("computed size = %d, want 24", desc.Size)
	}
	if desc.Alignment != 8 {
		t.Fatalf("computed alignment = %d, want 8", desc.Alignment)
	}
}

func TestClassifyArgumentAlignedOversizeByRef(t *testing.T) {
	alignedMember := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      16,
		Alignment: 16,
		Members: []*types.TypeDescriptor{
			types.UInt64TypeDescriptor,
			types.UInt64TypeDescriptor,
		},
	}

	desc := &types.TypeDescriptor{
		Kind: types.StructType,
		Members: []*types.TypeDescriptor{
			types.UInt8TypeDescriptor,
			alignedMember,
		},
	}

	class := classifyArgumentARM64(desc, types.UnixCallingConvention)
	if class.GPRCount != 1 || class.FPRCount != 0 {
		t.Fatalf("classifyArgumentARM64 aligned oversize = (GPR=%d,FPR=%d), want (1,0)",
			class.GPRCount, class.FPRCount)
	}
	if desc.Size != 32 {
		t.Fatalf("computed size = %d, want 32", desc.Size)
	}
	if desc.Alignment != 16 {
		t.Fatalf("computed alignment = %d, want 16", desc.Alignment)
	}
}

func TestClassifyReturnLargeStructSizeUnknown(t *testing.T) {
	desc := &types.TypeDescriptor{
		Kind: types.StructType,
		Members: []*types.TypeDescriptor{
			types.UInt64TypeDescriptor,
			types.UInt64TypeDescriptor,
			types.UInt64TypeDescriptor,
		},
	}

	ret := classifyReturnARM64(desc, types.UnixCallingConvention)
	if ret&types.ReturnViaPointer == 0 {
		t.Fatalf("classifyReturnARM64 large struct did not use sret")
	}
}
