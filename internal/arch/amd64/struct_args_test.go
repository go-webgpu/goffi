//go:build amd64

package amd64

import (
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// floatPairTypeAligned returns a {float32, float32} descriptor with its size
// and alignment set, as PrepareCallInterface leaves a nested member.
func floatPairTypeAligned() *types.TypeDescriptor {
	return &types.TypeDescriptor{
		Size: 8, Alignment: 4, Kind: types.StructType,
		Members: []*types.TypeDescriptor{types.FloatTypeDescriptor, types.FloatTypeDescriptor},
	}
}

// --- classifyEightbyte ---

func TestClassifyEightbyte(t *testing.T) {
	// {uint32, uint32}: both in eightbyte [0,8) → INTEGER
	intPairType := &types.TypeDescriptor{
		Size: 8, Kind: types.StructType,
		Members: []*types.TypeDescriptor{
			types.UInt32TypeDescriptor,
			types.UInt32TypeDescriptor,
		},
	}
	// {float32, float32}: both in eightbyte [0,8) → SSE
	floatPairType := &types.TypeDescriptor{
		Size: 8, Kind: types.StructType,
		Members: []*types.TypeDescriptor{
			types.FloatTypeDescriptor,
			types.FloatTypeDescriptor,
		},
	}
	// {float32, uint32}: float at offset 0, int at offset 4 — INTEGER wins
	mixedType := &types.TypeDescriptor{
		Size: 8, Kind: types.StructType,
		Members: []*types.TypeDescriptor{
			types.FloatTypeDescriptor,
			types.UInt32TypeDescriptor,
		},
	}
	// {double, uint64}: double at [0,8), uint64 at [8,16)
	splitType := &types.TypeDescriptor{
		Size: 16, Kind: types.StructType,
		Members: []*types.TypeDescriptor{
			types.DoubleTypeDescriptor,
			types.UInt64TypeDescriptor,
		},
	}

	// {{float32, float32}, {float32, float32}}: nested members are classified
	// by their offsets, so both eightbytes are SSE (psABI §3.2.3).
	nestedFloatsType := &types.TypeDescriptor{
		Size: 16, Alignment: 4, Kind: types.StructType,
		Members: []*types.TypeDescriptor{floatPairTypeAligned(), floatPairTypeAligned()},
	}
	// {int64, {float32, float32}}: INTEGER, then a nested SSE eightbyte.
	intThenNestedType := &types.TypeDescriptor{
		Size: 16, Alignment: 8, Kind: types.StructType,
		Members: []*types.TypeDescriptor{types.SInt64TypeDescriptor, floatPairTypeAligned()},
	}
	// {float32, {float32, int32}}: the nested int32 lands in the second eightbyte.
	nestedMixedType := &types.TypeDescriptor{
		Size: 12, Alignment: 4, Kind: types.StructType,
		Members: []*types.TypeDescriptor{types.FloatTypeDescriptor, {
			Size: 8, Alignment: 4, Kind: types.StructType,
			Members: []*types.TypeDescriptor{types.FloatTypeDescriptor, types.SInt32TypeDescriptor},
		}},
	}

	tests := []struct {
		name     string
		typ      *types.TypeDescriptor
		startOff uintptr
		endOff   uintptr
		wantSSE  bool
	}{
		{"empty members", &types.TypeDescriptor{Kind: types.StructType}, 0, 8, false},
		{"pointer only", &types.TypeDescriptor{Size: 8, Alignment: 8, Kind: types.StructType,
			Members: []*types.TypeDescriptor{types.PointerTypeDescriptor}}, 0, 8, false},
		{"nested floats [0,8)", nestedFloatsType, 0, 8, true},
		{"nested floats [8,16)", nestedFloatsType, 8, 16, true},
		{"int then nested floats [0,8)", intThenNestedType, 0, 8, false},
		{"int then nested floats [8,16)", intThenNestedType, 8, 16, true},
		{"nested mixed [0,8)", nestedMixedType, 0, 8, true},
		{"nested mixed [8,12)", nestedMixedType, 8, 12, false},
		{"int pair [0,8)", intPairType, 0, 8, false},
		{"float pair [0,8)", floatPairType, 0, 8, true},
		{"mixed float+int [0,8) INTEGER wins", mixedType, 0, 8, false},
		{"split: double in [0,8)", splitType, 0, 8, true},
		{"split: uint64 in [8,16)", splitType, 8, 16, false},
		// No field in range → false
		{"no field in range", intPairType, 16, 24, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyEightbyte(tt.typ, tt.startOff, tt.endOff)
			if got != tt.wantSSE {
				t.Errorf("classifyEightbyte(startOff=%d, endOff=%d) = %v, want %v",
					tt.startOff, tt.endOff, got, tt.wantSSE)
			}
		})
	}
}

// --- struct value reading correctness (unit-level, no C call) ---
// These tests verify that the byte patterns we'd read from a struct pointer
// are the values we expect — ensuring the sized-read logic is sound.

func TestStructValueRead1Byte(t *testing.T) {
	var v uint8 = 0xAB
	ptr := unsafe.Pointer(&v)
	got := uintptr(*(*uint8)(ptr))
	if got != uintptr(v) {
		t.Errorf("1-byte read: got %x, want %x", got, v)
	}
}

func TestStructValueRead2Byte(t *testing.T) {
	var v uint16 = 0xBEEF
	ptr := unsafe.Pointer(&v)
	got := uintptr(*(*uint16)(ptr))
	if got != uintptr(v) {
		t.Errorf("2-byte read: got %x, want %x", got, v)
	}
}

func TestStructValueRead4Byte(t *testing.T) {
	var v uint32 = 0xDEADBEEF
	ptr := unsafe.Pointer(&v)
	got := uintptr(*(*uint32)(ptr))
	if got != uintptr(v) {
		t.Errorf("4-byte read: got %x, want %x", got, v)
	}
}

func TestStructValueRead8Byte(t *testing.T) {
	var v uint64 = 0x0102030405060708
	ptr := unsafe.Pointer(&v)
	got := *(*uintptr)(ptr)
	if got != uintptr(v) {
		t.Errorf("8-byte read: got %x, want %x", got, v)
	}
}

// TestStructValueReadSecondEightbyte verifies reading the second half of a 16-byte struct.
func TestStructValueReadSecondEightbyte(t *testing.T) {
	var buf [16]byte
	for i := range buf {
		buf[i] = byte(i + 1)
	}
	firstPtr := unsafe.Pointer(&buf[0])
	secondPtr := unsafe.Add(firstPtr, 8)

	first := *(*uintptr)(firstPtr)
	second := *(*uintptr)(secondPtr)

	// On little-endian: first uint64 = bytes 0..7 = 0x0807060504030201
	expectedFirst := uint64(0x0807060504030201)
	expectedSecond := uint64(0x100F0E0D0C0B0A09)

	if uint64(first) != expectedFirst {
		t.Errorf("first eightbyte: got %x, want %x", first, expectedFirst)
	}
	if uint64(second) != expectedSecond {
		t.Errorf("second eightbyte: got %x, want %x", second, expectedSecond)
	}
}

// --- classifyArgumentAMD64 struct cases ---

func TestClassifyArgumentAMD64Structs(t *testing.T) {
	abi := types.UnixCallingConvention

	tests := []struct {
		name    string
		typ     *types.TypeDescriptor
		wantGPR int
		wantSSE int
	}{
		{
			// {uint32, uint32} 8B: single eightbyte, all INTEGER → 1 GPR
			"Struct8B_twoInts",
			&types.TypeDescriptor{Size: 8, Kind: types.StructType, Members: []*types.TypeDescriptor{
				types.UInt32TypeDescriptor,
				types.UInt32TypeDescriptor,
			}},
			1, 0,
		},
		{
			// {float32, float32} 8B: single eightbyte, all SSE → 1 SSE
			"Struct8B_twoFloats",
			&types.TypeDescriptor{Size: 8, Kind: types.StructType, Members: []*types.TypeDescriptor{
				types.FloatTypeDescriptor,
				types.FloatTypeDescriptor,
			}},
			0, 1,
		},
		{
			// {float32, uint32} 8B: INTEGER wins → 1 GPR
			"Struct8B_floatAndInt_INTEGERwins",
			&types.TypeDescriptor{Size: 8, Kind: types.StructType, Members: []*types.TypeDescriptor{
				types.FloatTypeDescriptor,
				types.UInt32TypeDescriptor,
			}},
			1, 0,
		},
		{
			// {double, uint64} 16B: eightbyte0=SSE, eightbyte1=INTEGER → 1 GPR + 1 SSE
			"Struct16B_doubleAndUint64",
			&types.TypeDescriptor{Size: 16, Kind: types.StructType, Members: []*types.TypeDescriptor{
				types.DoubleTypeDescriptor,
				types.UInt64TypeDescriptor,
			}},
			1, 1,
		},
		{
			// {double, double} 16B: both eightbytes SSE → 0 GPR + 2 SSE
			"Struct16B_twoDoubles",
			&types.TypeDescriptor{Size: 16, Kind: types.StructType, Members: []*types.TypeDescriptor{
				types.DoubleTypeDescriptor,
				types.DoubleTypeDescriptor,
			}},
			0, 2,
		},
		{
			// 32B non-HFA: MEMORY class → 0 GPR + 0 SSE
			"Struct32B_memoryClass",
			&types.TypeDescriptor{Size: 32, Kind: types.StructType},
			0, 0,
		},
		{
			// 17B: MEMORY class (> 16) → 0 GPR + 0 SSE
			"Struct17B_memoryClass",
			&types.TypeDescriptor{Size: 17, Kind: types.StructType},
			0, 0,
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
