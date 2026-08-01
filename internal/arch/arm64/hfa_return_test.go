//go:build arm64

package arm64

import (
	"math"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// TestHandleHFAReturn_Checkptr verifies that handleHFAReturn does not
// cast rvalue to an oversized array. Before the fix, (*[4]float64)(rvalue)
// on a 2-element HFA (CGSize = 16 bytes) triggered checkptr:
//
//	"converted pointer straddles multiple allocations"
//
// This test allocates exact-sized buffers and writes HFA return values.
// Under -race (which enables checkptr), the old code would crash here.
func TestHandleHFAReturn_Checkptr(t *testing.T) {
	impl := &Implementation{}

	tests := []struct {
		name     string
		flags    int
		count    int
		isFloat  bool
		values   [4]uint64
		expected []float64
	}{
		{
			name:     "HFA2 float64 (CGSize/CGPoint)",
			flags:    types.ReturnHFA2 | types.ReturnInXMM64,
			count:    2,
			values:   [4]uint64{math.Float64bits(1.5), math.Float64bits(2.5), 0, 0},
			expected: []float64{1.5, 2.5},
		},
		{
			name:     "HFA3 float64",
			flags:    types.ReturnHFA3 | types.ReturnInXMM64,
			count:    3,
			values:   [4]uint64{math.Float64bits(10.0), math.Float64bits(20.0), math.Float64bits(30.0), 0},
			expected: []float64{10.0, 20.0, 30.0},
		},
		{
			name:     "HFA4 float64",
			flags:    types.ReturnHFA4 | types.ReturnInXMM64,
			count:    4,
			values:   [4]uint64{math.Float64bits(1.0), math.Float64bits(2.0), math.Float64bits(3.0), math.Float64bits(4.0)},
			expected: []float64{1.0, 2.0, 3.0, 4.0},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Allocate exact-sized buffer matching real usage.
			// checkptr validates cast target size against allocation.
			buf := make([]float64, tc.count)
			rvalue := unsafe.Pointer(&buf[0])

			cif := &types.CallInterface{
				Flags: tc.flags,
				ReturnType: &types.TypeDescriptor{
					Kind: types.StructType,
					Members: func() []*types.TypeDescriptor {
						m := make([]*types.TypeDescriptor, tc.count)
						for i := range m {
							m[i] = types.DoubleTypeDescriptor
						}
						return m
					}(),
				},
			}

			err := impl.handleHFAReturn(cif, rvalue, tc.values)
			if err != nil {
				t.Fatalf("handleHFAReturn error: %v", err)
			}

			for i, want := range tc.expected {
				if buf[i] != want {
					t.Errorf("element[%d] = %f, want %f", i, buf[i], want)
				}
			}
		})
	}
}

// TestHandleHFAReturn_Float32 tests float32 HFA returns with exact-sized buffers.
func TestHandleHFAReturn_Float32(t *testing.T) {
	impl := &Implementation{}

	buf := make([]float32, 2)
	rvalue := unsafe.Pointer(&buf[0])

	cif := &types.CallInterface{
		Flags: types.ReturnHFA2 | types.ReturnInXMM32,
		ReturnType: &types.TypeDescriptor{
			Kind: types.StructType,
			Members: []*types.TypeDescriptor{
				types.FloatTypeDescriptor,
				types.FloatTypeDescriptor,
			},
		},
	}

	fret := [4]uint64{
		uint64(math.Float32bits(3.14)),
		uint64(math.Float32bits(2.71)),
		0, 0,
	}

	err := impl.handleHFAReturn(cif, rvalue, fret)
	if err != nil {
		t.Fatalf("handleHFAReturn error: %v", err)
	}

	if buf[0] != 3.14 {
		t.Errorf("float32[0] = %f, want 3.14", buf[0])
	}
	if buf[1] != 2.71 {
		t.Errorf("float32[1] = %f, want 2.71", buf[1])
	}
}
