//go:build arm64

package arm64

import (
	"math"
	"reflect"
	"testing"
	"unsafe"

	_ "github.com/go-webgpu/goffi/internal/fakecgo"
	"github.com/go-webgpu/goffi/types"
)

type abiCapture struct {
	GPR [8]uintptr
	FPR [8]uint64
	X8  uintptr
}

// captureABI is implemented in abi_capture_test.s.
//
//go:noescape
func captureABI(out *abiCapture)

func captureCall(t *testing.T, argTypes []*types.TypeDescriptor, args []unsafe.Pointer) abiCapture {
	t.Helper()
	var out abiCapture

	argTypes = append([]*types.TypeDescriptor{types.PointerTypeDescriptor}, argTypes...)
	outPtr := uintptr(unsafe.Pointer(&out))
	args = append([]unsafe.Pointer{unsafe.Pointer(&outPtr)}, args...)

	cif := &types.CallInterface{
		ArgCount:   len(argTypes),
		ArgTypes:   argTypes,
		ReturnType: types.VoidTypeDescriptor,
	}

	fnPtr := unsafe.Pointer(reflect.ValueOf(captureABI).Pointer())
	if fnPtr == nil {
		t.Fatalf("captureABI pointer is nil")
	}

	var impl Implementation
	if err := impl.Execute(cif, fnPtr, nil, args); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	return out
}

func TestExecuteCaptureRegistersSimple(t *testing.T) {
	intPatterns := []uint64{
		0x0,
		0x1,
		0xFFFFFFFFFFFFFFFF,
		0x0123456789ABCDEF,
		0xFEDCBA9876543210,
		0x1111111122222222,
		0x3333333344444444,
		0x5555555566666666,
		0x7777777788888888,
	}
	floatBits := []uint64{
		0x0000000000000000, // +0
		0x8000000000000000, // -0
		0x3FF0000000000000, // 1.0
		0xBFF0000000000000, // -1.0
		0x4008000000000000, // 3.0
		0xC008000000000000, // -3.0
		0x7FEFFFFFFFFFFFFF, // max finite
		0x0010000000000000, // min normal
	}

	for i := 0; i+3 < len(intPatterns); i++ {
		a1 := intPatterns[i]
		a2 := intPatterns[i+1]
		a3 := intPatterns[i+2]
		a4 := intPatterns[i+3]
		f0 := math.Float64frombits(floatBits[i%len(floatBits)])
		f1 := math.Float64frombits(floatBits[(i+3)%len(floatBits)])

		out := captureCall(
			t,
			[]*types.TypeDescriptor{
				types.UInt64TypeDescriptor,
				types.UInt64TypeDescriptor,
				types.UInt64TypeDescriptor,
				types.UInt64TypeDescriptor,
				types.DoubleTypeDescriptor,
				types.DoubleTypeDescriptor,
			},
			[]unsafe.Pointer{
				unsafe.Pointer(&a1),
				unsafe.Pointer(&a2),
				unsafe.Pointer(&a3),
				unsafe.Pointer(&a4),
				unsafe.Pointer(&f0),
				unsafe.Pointer(&f1),
			},
		)

		if out.GPR[1] != uintptr(a1) || out.GPR[2] != uintptr(a2) ||
			out.GPR[3] != uintptr(a3) || out.GPR[4] != uintptr(a4) {
			t.Fatalf("GPR mismatch: %x %x %x %x", out.GPR[1], out.GPR[2], out.GPR[3], out.GPR[4])
		}
		if out.FPR[0] != math.Float64bits(f0) || out.FPR[1] != math.Float64bits(f1) {
			t.Fatalf("FPR mismatch: 0x%x 0x%x", out.FPR[0], out.FPR[1])
		}
	}
}

func TestExecuteCaptureStructHFA(t *testing.T) {
	type Vec4 struct {
		A float64
		B float64
		C float64
		D float64
	}

	desc := &types.TypeDescriptor{
		Kind:      types.StructType,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			types.DoubleTypeDescriptor,
			types.DoubleTypeDescriptor,
			types.DoubleTypeDescriptor,
			types.DoubleTypeDescriptor,
		},
	}

	patterns := []Vec4{
		{A: 1.0, B: 2.0, C: 3.0, D: 4.0},
		{A: -1.0, B: -2.0, C: 0.5, D: 0.25},
		{A: math.Float64frombits(0x7FEFFFFFFFFFFFFF), B: 0, C: -0, D: 1},
	}

	for _, val := range patterns {
		out := captureCall(
			t,
			[]*types.TypeDescriptor{desc},
			[]unsafe.Pointer{unsafe.Pointer(&val)},
		)

		if out.GPR[1] != 0 {
			t.Fatalf("expected no GPR usage for HFA, got 0x%x", out.GPR[1])
		}
		if out.FPR[0] != math.Float64bits(val.A) ||
			out.FPR[1] != math.Float64bits(val.B) ||
			out.FPR[2] != math.Float64bits(val.C) ||
			out.FPR[3] != math.Float64bits(val.D) {
			t.Fatalf("unexpected FPR contents for HFA")
		}
	}
}

func TestExecuteCaptureStructMixedSmall(t *testing.T) {
	type Mixed struct {
		A uint32
		B float32
	}

	desc := &types.TypeDescriptor{
		Kind:      types.StructType,
		Alignment: 4,
		Members: []*types.TypeDescriptor{
			types.UInt32TypeDescriptor,
			types.FloatTypeDescriptor,
		},
	}

	patterns := []Mixed{
		{A: 0x11223344, B: 1.5},
		{A: 0xFFFFFFFF, B: -2.0},
		{A: 0x0, B: math.Float32frombits(0x7F7FFFFF)},
	}

	for _, val := range patterns {
		out := captureCall(
			t,
			[]*types.TypeDescriptor{desc},
			[]unsafe.Pointer{unsafe.Pointer(&val)},
		)

		want := uint64(val.A) | (uint64(math.Float32bits(val.B)) << 32)
		if uint64(out.GPR[1]) != want {
			t.Fatalf("packed GPR mismatch: 0x%x, want 0x%x", uint64(out.GPR[1]), want)
		}
		if out.FPR[0] != 0 {
			t.Fatalf("expected no FPR usage for mixed small struct, got 0x%x", out.FPR[0])
		}
	}
}

func TestExecuteCaptureStructByRefLarge(t *testing.T) {
	type Large struct {
		A uint64
		B uint64
		C uint64
	}

	desc := &types.TypeDescriptor{
		Kind: types.StructType,
		Members: []*types.TypeDescriptor{
			types.UInt64TypeDescriptor,
			types.UInt64TypeDescriptor,
			types.UInt64TypeDescriptor,
		},
	}

	patterns := []Large{
		{A: 1, B: 2, C: 3},
		{A: 0xFFFFFFFFFFFFFFFF, B: 0, C: 0xAAAAAAAAAAAAAAAA},
		{A: 0x0123456789ABCDEF, B: 0xFEDCBA9876543210, C: 0},
	}

	for _, val := range patterns {
		out := captureCall(
			t,
			[]*types.TypeDescriptor{desc},
			[]unsafe.Pointer{unsafe.Pointer(&val)},
		)

		if out.GPR[1] != uintptr(unsafe.Pointer(&val)) {
			t.Fatalf("by-ref GPR mismatch: 0x%x, want 0x%x", out.GPR[1], uintptr(unsafe.Pointer(&val)))
		}
	}
}
