//go:build arm64

package arm64

import (
	"testing"

	"github.com/go-webgpu/goffi/types"
)

// TestClassifyReturnHFA tests HFA (Homogeneous Floating-point Aggregate) return classification.
// This is critical for NSRect (4 x float64) and similar Objective-C types on macOS ARM64.
func TestClassifyReturnHFA(t *testing.T) {
	tests := []struct {
		name     string
		typ      *types.TypeDescriptor
		expected int
	}{
		{
			name:     "single float",
			typ:      types.FloatTypeDescriptor,
			expected: types.ReturnInXMM32,
		},
		{
			name:     "single double",
			typ:      types.DoubleTypeDescriptor,
			expected: types.ReturnInXMM64,
		},
		{
			name: "HFA 2 doubles (CGPoint)",
			typ: &types.TypeDescriptor{
				Kind:      types.StructType,
				Size:      16,
				Alignment: 8,
				Members: []*types.TypeDescriptor{
					types.DoubleTypeDescriptor,
					types.DoubleTypeDescriptor,
				},
			},
			expected: types.ReturnHFA2 | types.ReturnInXMM64,
		},
		{
			name: "HFA 3 doubles",
			typ: &types.TypeDescriptor{
				Kind:      types.StructType,
				Size:      24,
				Alignment: 8,
				Members: []*types.TypeDescriptor{
					types.DoubleTypeDescriptor,
					types.DoubleTypeDescriptor,
					types.DoubleTypeDescriptor,
				},
			},
			expected: types.ReturnHFA3 | types.ReturnInXMM64,
		},
		{
			name: "HFA 4 doubles (NSRect)",
			typ: &types.TypeDescriptor{
				Kind:      types.StructType,
				Size:      32,
				Alignment: 8,
				Members: []*types.TypeDescriptor{
					types.DoubleTypeDescriptor,
					types.DoubleTypeDescriptor,
					types.DoubleTypeDescriptor,
					types.DoubleTypeDescriptor,
				},
			},
			expected: types.ReturnHFA4 | types.ReturnInXMM64,
		},
		{
			name: "nested HFA 4 doubles (CGRect)",
			typ: &types.TypeDescriptor{
				Kind:      types.StructType,
				Size:      32,
				Alignment: 8,
				Members: []*types.TypeDescriptor{
					{
						Kind:      types.StructType,
						Alignment: 8,
						Members: []*types.TypeDescriptor{
							types.DoubleTypeDescriptor,
							types.DoubleTypeDescriptor,
						},
					},
					{
						Kind:      types.StructType,
						Alignment: 8,
						Members: []*types.TypeDescriptor{
							types.DoubleTypeDescriptor,
							types.DoubleTypeDescriptor,
						},
					},
				},
			},
			expected: types.ReturnHFA4 | types.ReturnInXMM64,
		},
		{
			name: "HFA 4 floats (CGRect float)",
			typ: &types.TypeDescriptor{
				Kind:      types.StructType,
				Size:      16,
				Alignment: 4,
				Members: []*types.TypeDescriptor{
					types.FloatTypeDescriptor,
					types.FloatTypeDescriptor,
					types.FloatTypeDescriptor,
					types.FloatTypeDescriptor,
				},
			},
			expected: types.ReturnHFA4 | types.ReturnInXMM32,
		},
		{
			name: "small struct (not HFA)",
			typ: &types.TypeDescriptor{
				Kind:      types.StructType,
				Size:      8,
				Alignment: 8,
				Members: []*types.TypeDescriptor{
					types.UInt64TypeDescriptor,
				},
			},
			expected: types.ReturnInt64,
		},
		{
			name: "16-byte struct (not HFA)",
			typ: &types.TypeDescriptor{
				Kind:      types.StructType,
				Size:      16,
				Alignment: 8,
				Members: []*types.TypeDescriptor{
					types.UInt64TypeDescriptor,
					types.UInt64TypeDescriptor,
				},
			},
			expected: types.ReturnInt64,
		},
		{
			name: "large struct (sret)",
			typ: &types.TypeDescriptor{
				Kind:      types.StructType,
				Size:      64,
				Alignment: 8,
				Members: []*types.TypeDescriptor{
					types.UInt64TypeDescriptor,
					types.UInt64TypeDescriptor,
					types.UInt64TypeDescriptor,
					types.UInt64TypeDescriptor,
					types.UInt64TypeDescriptor,
					types.UInt64TypeDescriptor,
					types.UInt64TypeDescriptor,
					types.UInt64TypeDescriptor,
				},
			},
			expected: types.ReturnViaPointer | types.ReturnVoid,
		},
		{
			name:     "void",
			typ:      types.VoidTypeDescriptor,
			expected: types.ReturnVoid,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := classifyReturnARM64(tc.typ, types.UnixCallingConvention)
			if result != tc.expected {
				t.Errorf("classifyReturnARM64(%s) = %d (0x%x), want %d (0x%x)",
					tc.name, result, result, tc.expected, tc.expected)
			}
		})
	}
}

// TestIsHomogeneousFloatAggregate tests HFA detection.
func TestIsHomogeneousFloatAggregate(t *testing.T) {
	tests := []struct {
		name     string
		typ      *types.TypeDescriptor
		isHFA    bool
		hfaCount int
	}{
		{
			name: "4 doubles (NSRect)",
			typ: &types.TypeDescriptor{
				Kind: types.StructType,
				Members: []*types.TypeDescriptor{
					types.DoubleTypeDescriptor,
					types.DoubleTypeDescriptor,
					types.DoubleTypeDescriptor,
					types.DoubleTypeDescriptor,
				},
			},
			isHFA:    true,
			hfaCount: 4,
		},
		{
			name: "2 doubles (CGPoint)",
			typ: &types.TypeDescriptor{
				Kind: types.StructType,
				Members: []*types.TypeDescriptor{
					types.DoubleTypeDescriptor,
					types.DoubleTypeDescriptor,
				},
			},
			isHFA:    true,
			hfaCount: 2,
		},
		{
			name: "nested 2 doubles (CGSize)",
			typ: &types.TypeDescriptor{
				Kind: types.StructType,
				Members: []*types.TypeDescriptor{
					{
						Kind: types.StructType,
						Members: []*types.TypeDescriptor{
							types.DoubleTypeDescriptor,
							types.DoubleTypeDescriptor,
						},
					},
				},
			},
			isHFA:    true,
			hfaCount: 2,
		},
		{
			name: "nested 4 doubles (CGRect)",
			typ: &types.TypeDescriptor{
				Kind: types.StructType,
				Members: []*types.TypeDescriptor{
					{
						Kind: types.StructType,
						Members: []*types.TypeDescriptor{
							types.DoubleTypeDescriptor,
							types.DoubleTypeDescriptor,
						},
					},
					{
						Kind: types.StructType,
						Members: []*types.TypeDescriptor{
							types.DoubleTypeDescriptor,
							types.DoubleTypeDescriptor,
						},
					},
				},
			},
			isHFA:    true,
			hfaCount: 4,
		},
		{
			name: "mixed types (not HFA)",
			typ: &types.TypeDescriptor{
				Kind: types.StructType,
				Members: []*types.TypeDescriptor{
					types.DoubleTypeDescriptor,
					types.UInt64TypeDescriptor,
				},
			},
			isHFA:    false,
			hfaCount: 0,
		},
		{
			name: "nested mixed types (not HFA)",
			typ: &types.TypeDescriptor{
				Kind: types.StructType,
				Members: []*types.TypeDescriptor{
					{
						Kind: types.StructType,
						Members: []*types.TypeDescriptor{
							types.DoubleTypeDescriptor,
							types.UInt64TypeDescriptor,
						},
					},
				},
			},
			isHFA:    false,
			hfaCount: 0,
		},
		{
			name: "5 doubles (too many for HFA)",
			typ: &types.TypeDescriptor{
				Kind: types.StructType,
				Members: []*types.TypeDescriptor{
					types.DoubleTypeDescriptor,
					types.DoubleTypeDescriptor,
					types.DoubleTypeDescriptor,
					types.DoubleTypeDescriptor,
					types.DoubleTypeDescriptor,
				},
			},
			isHFA:    false,
			hfaCount: 0,
		},
		{
			name: "empty struct",
			typ: &types.TypeDescriptor{
				Kind:    types.StructType,
				Members: []*types.TypeDescriptor{},
			},
			isHFA:    false,
			hfaCount: 0,
		},
		{
			name:     "not a struct",
			typ:      types.UInt64TypeDescriptor,
			isHFA:    false,
			hfaCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isHFA, hfaCount, _ := isHomogeneousFloatAggregate(tc.typ)
			if isHFA != tc.isHFA || hfaCount != tc.hfaCount {
				t.Errorf("isHomogeneousFloatAggregate(%s) = (%v, %d), want (%v, %d)",
					tc.name, isHFA, hfaCount, tc.isHFA, tc.hfaCount)
			}
		})
	}
}
