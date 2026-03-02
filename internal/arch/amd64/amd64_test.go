//go:build amd64

package amd64

import (
	"math"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

func TestAlign(t *testing.T) {
	impl := &Implementation{}
	tests := []struct {
		value, alignment, want uintptr
	}{
		{0, 8, 0},
		{1, 8, 8},
		{7, 8, 8},
		{8, 8, 8},
		{9, 8, 16},
		{15, 16, 16},
		{16, 16, 16},
		{17, 16, 32},
		{1, 1, 1},
		{3, 4, 4},
		{4, 4, 4},
	}
	for _, tt := range tests {
		got := impl.align(tt.value, tt.alignment)
		if got != tt.want {
			t.Errorf("align(%d, %d) = %d, want %d", tt.value, tt.alignment, got, tt.want)
		}
	}
}

func TestClassifyReturnAMD64(t *testing.T) {
	abi := types.UnixCallingConvention

	tests := []struct {
		name string
		typ  *types.TypeDescriptor
		want int
	}{
		{"Void", types.VoidTypeDescriptor, types.ReturnVoid},
		{"Float", types.FloatTypeDescriptor, types.ReturnInXMM32},
		{"Double", types.DoubleTypeDescriptor, types.ReturnInXMM64},
		{"UInt8", types.UInt8TypeDescriptor, types.ReturnInt64},
		{"SInt8", types.SInt8TypeDescriptor, types.ReturnInt64},
		{"UInt16", types.UInt16TypeDescriptor, types.ReturnInt64},
		{"SInt16", types.SInt16TypeDescriptor, types.ReturnInt64},
		{"UInt32", types.UInt32TypeDescriptor, types.ReturnInt64},
		{"SInt32", types.SInt32TypeDescriptor, types.ReturnInt64},
		{"UInt64", types.UInt64TypeDescriptor, types.ReturnInt64},
		{"SInt64", types.SInt64TypeDescriptor, types.ReturnInt64},
		{"Int", types.IntTypeDescriptor, types.ReturnInt64},
		{"Pointer", types.PointerTypeDescriptor, types.ReturnInt64},
		{"Struct1B", &types.TypeDescriptor{Size: 1, Kind: types.StructType}, types.ReturnSInt8},
		{"Struct2B", &types.TypeDescriptor{Size: 2, Kind: types.StructType}, types.ReturnSInt16},
		{"Struct4B", &types.TypeDescriptor{Size: 4, Kind: types.StructType}, types.ReturnSInt32},
		{"Struct8B", &types.TypeDescriptor{Size: 8, Kind: types.StructType}, types.ReturnInt64},
		{"Struct24B", &types.TypeDescriptor{Size: 24, Kind: types.StructType}, types.ReturnViaPointer | types.ReturnVoid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyReturnAMD64(tt.typ, abi)
			if got != tt.want {
				t.Errorf("classifyReturnAMD64(%s) = %d, want %d", tt.name, got, tt.want)
			}
		})
	}
}

func TestClassifyArgumentAMD64(t *testing.T) {
	abi := types.UnixCallingConvention

	tests := []struct {
		name    string
		typ     *types.TypeDescriptor
		wantGPR int
		wantSSE int
	}{
		{"Int", types.IntTypeDescriptor, 1, 0},
		{"UInt64", types.UInt64TypeDescriptor, 1, 0},
		{"Pointer", types.PointerTypeDescriptor, 1, 0},
		{"UInt8", types.UInt8TypeDescriptor, 1, 0},
		{"Float", types.FloatTypeDescriptor, 0, 1},
		{"Double", types.DoubleTypeDescriptor, 0, 1},
		{
			"Struct16B_noFloat",
			&types.TypeDescriptor{Size: 16, Kind: types.StructType, Members: []*types.TypeDescriptor{
				types.UInt64TypeDescriptor,
				types.UInt64TypeDescriptor,
			}},
			2, 0,
		},
		{
			"Struct16B_withFloat",
			&types.TypeDescriptor{Size: 16, Kind: types.StructType, Members: []*types.TypeDescriptor{
				types.FloatTypeDescriptor,
				types.UInt64TypeDescriptor,
			}},
			1, 1,
		},
		{
			"Struct24B_large",
			&types.TypeDescriptor{Size: 24, Kind: types.StructType},
			3, 0,
		},
		{
			"Struct8B_withDouble",
			&types.TypeDescriptor{Size: 8, Kind: types.StructType, Members: []*types.TypeDescriptor{
				types.DoubleTypeDescriptor,
			}},
			0, 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyArgumentAMD64(tt.typ, abi)
			if got.GPRCount != tt.wantGPR {
				t.Errorf("GPRCount = %d, want %d", got.GPRCount, tt.wantGPR)
			}
			if got.SSECount != tt.wantSSE {
				t.Errorf("SSECount = %d, want %d", got.SSECount, tt.wantSSE)
			}
		})
	}
}

func TestHandleReturn(t *testing.T) {
	impl := &Implementation{}

	t.Run("Void", func(t *testing.T) {
		cif := &types.CallInterface{ReturnType: types.VoidTypeDescriptor}
		err := impl.handleReturn(cif, nil, 0, 0)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("NilRvalue", func(t *testing.T) {
		cif := &types.CallInterface{ReturnType: types.UInt64TypeDescriptor}
		err := impl.handleReturn(cif, nil, 42, 0)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("UInt8", func(t *testing.T) {
		var result uint8
		cif := &types.CallInterface{ReturnType: types.UInt8TypeDescriptor}
		err := impl.handleReturn(cif, unsafe.Pointer(&result), 0xFF, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != 0xFF {
			t.Errorf("got %d, want 255", result)
		}
	})

	t.Run("SInt8", func(t *testing.T) {
		var result int8
		cif := &types.CallInterface{ReturnType: types.SInt8TypeDescriptor}
		err := impl.handleReturn(cif, unsafe.Pointer(&result), uint64(0xFE), 0) // -2
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != -2 {
			t.Errorf("got %d, want -2", result)
		}
	})

	t.Run("UInt16", func(t *testing.T) {
		var result uint16
		cif := &types.CallInterface{ReturnType: types.UInt16TypeDescriptor}
		err := impl.handleReturn(cif, unsafe.Pointer(&result), 0xBEEF, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != 0xBEEF {
			t.Errorf("got %d, want %d", result, 0xBEEF)
		}
	})

	t.Run("SInt16", func(t *testing.T) {
		var result int16
		cif := &types.CallInterface{ReturnType: types.SInt16TypeDescriptor}
		err := impl.handleReturn(cif, unsafe.Pointer(&result), uint64(0xFFFF), 0) // -1
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != -1 {
			t.Errorf("got %d, want -1", result)
		}
	})

	t.Run("UInt32", func(t *testing.T) {
		var result uint32
		cif := &types.CallInterface{ReturnType: types.UInt32TypeDescriptor}
		err := impl.handleReturn(cif, unsafe.Pointer(&result), 0xDEADBEEF, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != 0xDEADBEEF {
			t.Errorf("got %d, want %d", result, uint32(0xDEADBEEF))
		}
	})

	t.Run("SInt32", func(t *testing.T) {
		var result int32
		cif := &types.CallInterface{ReturnType: types.SInt32TypeDescriptor}
		err := impl.handleReturn(cif, unsafe.Pointer(&result), uint64(0xFFFFFFFF), 0) // -1
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != -1 {
			t.Errorf("got %d, want -1", result)
		}
	})

	t.Run("UInt64", func(t *testing.T) {
		var result uint64
		cif := &types.CallInterface{ReturnType: types.UInt64TypeDescriptor}
		err := impl.handleReturn(cif, unsafe.Pointer(&result), 0x123456789ABCDEF0, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != 0x123456789ABCDEF0 {
			t.Errorf("got %x, want %x", result, uint64(0x123456789ABCDEF0))
		}
	})

	t.Run("SInt64", func(t *testing.T) {
		var result uint64
		cif := &types.CallInterface{ReturnType: types.SInt64TypeDescriptor}
		err := impl.handleReturn(cif, unsafe.Pointer(&result), 42, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != 42 {
			t.Errorf("got %d, want 42", result)
		}
	})

	t.Run("Pointer", func(t *testing.T) {
		var result uint64
		cif := &types.CallInterface{ReturnType: types.PointerTypeDescriptor}
		err := impl.handleReturn(cif, unsafe.Pointer(&result), 0xCAFEBABE, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != 0xCAFEBABE {
			t.Errorf("got %x, want %x", result, uint64(0xCAFEBABE))
		}
	})

	t.Run("Float32", func(t *testing.T) {
		var result float32
		expected := float32(3.14)
		bits := uint64(math.Float32bits(expected))
		cif := &types.CallInterface{ReturnType: types.FloatTypeDescriptor}
		err := impl.handleReturn(cif, unsafe.Pointer(&result), bits, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("got %f, want %f", result, expected)
		}
	})

	t.Run("Float64", func(t *testing.T) {
		var result float64
		expected := 2.71828
		bits := math.Float64bits(expected)
		cif := &types.CallInterface{ReturnType: types.DoubleTypeDescriptor}
		err := impl.handleReturn(cif, unsafe.Pointer(&result), bits, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("got %f, want %f", result, expected)
		}
	})

	t.Run("StructLE8", func(t *testing.T) {
		var result uint64
		cif := &types.CallInterface{
			ReturnType: &types.TypeDescriptor{Size: 8, Kind: types.StructType},
		}
		err := impl.handleReturn(cif, unsafe.Pointer(&result), 0xDEADCAFE, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != 0xDEADCAFE {
			t.Errorf("got %x, want %x", result, uint64(0xDEADCAFE))
		}
	})

	t.Run("Struct9to16", func(t *testing.T) {
		// 12-byte struct: RAX=low 8 bytes, RDX=high 4 bytes
		var buf [16]byte
		cif := &types.CallInterface{
			ReturnType: &types.TypeDescriptor{Size: 12, Kind: types.StructType},
		}
		retVal := uint64(0x0807060504030201)
		retVal2 := uint64(0x0000000C0B0A09)
		err := impl.handleReturn(cif, unsafe.Pointer(&buf[0]), retVal, retVal2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// First 8 bytes from RAX
		got := *(*uint64)(unsafe.Pointer(&buf[0]))
		if got != retVal {
			t.Errorf("low 8 bytes: got %x, want %x", got, retVal)
		}
		// Next 4 bytes from RDX (remaining = 12-8 = 4)
		for i := 0; i < 4; i++ {
			expected := byte((retVal2 >> (8 * i)) & 0xFF)
			if buf[8+i] != expected {
				t.Errorf("buf[%d] = %x, want %x", 8+i, buf[8+i], expected)
			}
		}
	})

	t.Run("StructGT16_sret", func(t *testing.T) {
		// Structs > 16B are returned via sret pointer — handleReturn should be a no-op
		var buf [32]byte
		buf[0] = 0xAA // pre-fill to verify no overwrite
		cif := &types.CallInterface{
			ReturnType: &types.TypeDescriptor{Size: 24, Kind: types.StructType},
		}
		err := impl.handleReturn(cif, unsafe.Pointer(&buf[0]), 0, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf[0] != 0xAA {
			t.Error("sret buffer was unexpectedly modified")
		}
	})

	t.Run("ReturnViaPointer", func(t *testing.T) {
		var dummy uint64 = 42
		var result unsafe.Pointer
		cif := &types.CallInterface{
			ReturnType: types.PointerTypeDescriptor,
			Flags:      types.ReturnViaPointer,
		}
		err := impl.handleReturn(cif, unsafe.Pointer(&result), uint64(uintptr(unsafe.Pointer(&dummy))), 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != unsafe.Pointer(&dummy) {
			t.Errorf("got %v, want %v", result, unsafe.Pointer(&dummy))
		}
	})
}

func TestClassifyReturnViaInterface(t *testing.T) {
	impl := &Implementation{}
	got := impl.ClassifyReturn(types.FloatTypeDescriptor, types.UnixCallingConvention)
	if got != types.ReturnInXMM32 {
		t.Errorf("ClassifyReturn(Float) = %d, want %d", got, types.ReturnInXMM32)
	}
}

func TestClassifyArgumentViaInterface(t *testing.T) {
	impl := &Implementation{}
	got := impl.ClassifyArgument(types.DoubleTypeDescriptor, types.UnixCallingConvention)
	if got.GPRCount != 0 || got.SSECount != 1 {
		t.Errorf("ClassifyArgument(Double) = {GPR:%d, SSE:%d}, want {GPR:0, SSE:1}", got.GPRCount, got.SSECount)
	}
}
