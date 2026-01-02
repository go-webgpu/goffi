//go:build arm64

package arm64

import (
	"fmt"
	"math"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

func placeStruct(t *testing.T, desc *types.TypeDescriptor, ptr unsafe.Pointer) (gpr [8]uintptr, fpr [8]uint64, gprIdx, fprIdx int) {
	t.Helper()
	ok := placeStructRegisters(
		ptr,
		desc,
		func(v uint64) bool {
			if gprIdx >= 8 {
				return false
			}
			gpr[gprIdx] = uintptr(v)
			gprIdx++
			return true
		},
		func(v uint64) bool {
			if fprIdx >= 8 {
				return false
			}
			fpr[fprIdx] = v
			fprIdx++
			return true
		},
	)
	if !ok {
		t.Fatalf("placeStructRegisters failed")
	}
	return gpr, fpr, gprIdx, fprIdx
}

func TestPlaceStructRegistersNSSize(t *testing.T) {
	type NSSize struct {
		Width  float64
		Height float64
	}

	desc := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      16,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			types.DoubleTypeDescriptor,
			types.DoubleTypeDescriptor,
		},
	}

	val := NSSize{Width: 800, Height: 600}
	_, fpr, gprIdx, fprIdx := placeStruct(t, desc, unsafe.Pointer(&val))

	if gprIdx != 0 {
		t.Fatalf("expected no GPR usage, got %d", gprIdx)
	}
	if fprIdx != 2 {
		t.Fatalf("expected 2 FPRs, got %d", fprIdx)
	}
	if fpr[0] != math.Float64bits(val.Width) {
		t.Fatalf("fpr[0] = 0x%x, want 0x%x", fpr[0], math.Float64bits(val.Width))
	}
	if fpr[1] != math.Float64bits(val.Height) {
		t.Fatalf("fpr[1] = 0x%x, want 0x%x", fpr[1], math.Float64bits(val.Height))
	}
}

func TestPlaceStructRegistersMixedChunk(t *testing.T) {
	type Mixed struct {
		A uint32
		B float32
	}

	desc := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      8,
		Alignment: 4,
		Members: []*types.TypeDescriptor{
			types.UInt32TypeDescriptor,
			types.FloatTypeDescriptor,
		},
	}

	val := Mixed{A: 0x11223344, B: 1.5}
	gpr, _, gprIdx, fprIdx := placeStruct(t, desc, unsafe.Pointer(&val))

	if fprIdx != 0 {
		t.Fatalf("expected no FPR usage, got %d", fprIdx)
	}
	if gprIdx != 1 {
		t.Fatalf("expected 1 GPR usage, got %d", gprIdx)
	}
	want := uint64(val.A) | (uint64(math.Float32bits(val.B)) << 32)
	if uint64(gpr[0]) != want {
		t.Fatalf("gpr[0] = 0x%x, want 0x%x", uint64(gpr[0]), want)
	}
}

func TestPlaceStructRegistersIntThenDouble(t *testing.T) {
	type Mixed struct {
		A uint32
		B float64
	}

	desc := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      16,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			types.UInt32TypeDescriptor,
			types.DoubleTypeDescriptor,
		},
	}

	val := Mixed{A: 0xdeadbeef, B: 2.5}
	gpr, fpr, gprIdx, fprIdx := placeStruct(t, desc, unsafe.Pointer(&val))

	if gprIdx != 1 {
		t.Fatalf("expected 1 GPR usage, got %d", gprIdx)
	}
	if fprIdx != 1 {
		t.Fatalf("expected 1 FPR usage, got %d", fprIdx)
	}
	if uint64(gpr[0]) != uint64(val.A) {
		t.Fatalf("gpr[0] = 0x%x, want 0x%x", uint64(gpr[0]), uint64(val.A))
	}
	if fpr[0] != math.Float64bits(val.B) {
		t.Fatalf("fpr[0] = 0x%x, want 0x%x", fpr[0], math.Float64bits(val.B))
	}
}

func TestPlaceStructRegistersDoubleThenFloat(t *testing.T) {
	type Mixed struct {
		A float64
		B float32
	}

	desc := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      16,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			types.DoubleTypeDescriptor,
			types.FloatTypeDescriptor,
		},
	}

	val := Mixed{A: 1.25, B: 0.75}
	_, fpr, gprIdx, fprIdx := placeStruct(t, desc, unsafe.Pointer(&val))

	if gprIdx != 0 {
		t.Fatalf("expected no GPR usage, got %d", gprIdx)
	}
	if fprIdx != 2 {
		t.Fatalf("expected 2 FPR usage, got %d", fprIdx)
	}
	if fpr[0] != math.Float64bits(val.A) {
		t.Fatalf("fpr[0] = 0x%x, want 0x%x", fpr[0], math.Float64bits(val.A))
	}
	if fpr[1] != uint64(math.Float32bits(val.B)) {
		t.Fatalf("fpr[1] = 0x%x, want 0x%x", fpr[1], uint64(math.Float32bits(val.B)))
	}
}

func TestPlaceStructRegistersPaddingSplit(t *testing.T) {
	type Mixed struct {
		A uint32
		B uint64
	}

	desc := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      16,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			types.UInt32TypeDescriptor,
			types.UInt64TypeDescriptor,
		},
	}

	val := Mixed{A: 0x11223344, B: 0x5566778899aabbcc}
	gpr, _, gprIdx, fprIdx := placeStruct(t, desc, unsafe.Pointer(&val))

	if fprIdx != 0 {
		t.Fatalf("expected no FPR usage, got %d", fprIdx)
	}
	if gprIdx != 2 {
		t.Fatalf("expected 2 GPR usage, got %d", gprIdx)
	}
	if uint64(gpr[0]) != uint64(val.A) {
		t.Fatalf("gpr[0] = 0x%x, want 0x%x", uint64(gpr[0]), uint64(val.A))
	}
	if uint64(gpr[1]) != val.B {
		t.Fatalf("gpr[1] = 0x%x, want 0x%x", uint64(gpr[1]), val.B)
	}
}

func TestPlaceStructRegistersFloat32HFA(t *testing.T) {
	type Vec2 struct {
		X float32
		Y float32
	}

	desc := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      8,
		Alignment: 4,
		Members: []*types.TypeDescriptor{
			types.FloatTypeDescriptor,
			types.FloatTypeDescriptor,
		},
	}

	val := Vec2{X: 0.25, Y: 0.5}
	_, fpr, gprIdx, fprIdx := placeStruct(t, desc, unsafe.Pointer(&val))

	if gprIdx != 0 {
		t.Fatalf("expected no GPR usage, got %d", gprIdx)
	}
	if fprIdx != 2 {
		t.Fatalf("expected 2 FPRs, got %d", fprIdx)
	}
	if fpr[0] != uint64(math.Float32bits(val.X)) {
		t.Fatalf("fpr[0] = 0x%x, want 0x%x", fpr[0], uint64(math.Float32bits(val.X)))
	}
	if fpr[1] != uint64(math.Float32bits(val.Y)) {
		t.Fatalf("fpr[1] = 0x%x, want 0x%x", fpr[1], uint64(math.Float32bits(val.Y)))
	}
}

func TestPlaceStructRegistersNestedHFA(t *testing.T) {
	type CGPoint struct {
		X float64
		Y float64
	}
	type CGSize struct {
		W float64
		H float64
	}
	type CGRect struct {
		Origin CGPoint
		Size   CGSize
	}

	desc := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      32,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			{
				Kind:      types.StructType,
				Size:      16,
				Alignment: 8,
				Members: []*types.TypeDescriptor{
					types.DoubleTypeDescriptor,
					types.DoubleTypeDescriptor,
				},
			},
			{
				Kind:      types.StructType,
				Size:      16,
				Alignment: 8,
				Members: []*types.TypeDescriptor{
					types.DoubleTypeDescriptor,
					types.DoubleTypeDescriptor,
				},
			},
		},
	}

	val := CGRect{
		Origin: CGPoint{X: 1.25, Y: 2.5},
		Size:   CGSize{W: 3.75, H: 4.0},
	}
	_, fpr, gprIdx, fprIdx := placeStruct(t, desc, unsafe.Pointer(&val))

	if gprIdx != 0 {
		t.Fatalf("expected no GPR usage, got %d", gprIdx)
	}
	if fprIdx != 4 {
		t.Fatalf("expected 4 FPRs, got %d", fprIdx)
	}
	if fpr[0] != math.Float64bits(val.Origin.X) ||
		fpr[1] != math.Float64bits(val.Origin.Y) ||
		fpr[2] != math.Float64bits(val.Size.W) ||
		fpr[3] != math.Float64bits(val.Size.H) {
		t.Fatalf("unexpected FPR contents for nested HFA")
	}
}

func TestPlaceStructRegistersFailOnFPROverflow(t *testing.T) {
	type Vec4 struct {
		A float64
		B float64
		C float64
		D float64
	}

	desc := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      32,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			types.DoubleTypeDescriptor,
			types.DoubleTypeDescriptor,
			types.DoubleTypeDescriptor,
			types.DoubleTypeDescriptor,
		},
	}

	val := Vec4{A: 1, B: 2, C: 3, D: 4}

	var fprIdx int
	ok := placeStructRegisters(
		unsafe.Pointer(&val),
		desc,
		func(uint64) bool { return true },
		func(uint64) bool {
			if fprIdx >= 1 {
				return false
			}
			fprIdx++
			return true
		},
	)

	if ok {
		t.Fatalf("expected placeStructRegisters to fail when FPRs are exhausted")
	}
}

func TestCountStructRegUsageMixed(t *testing.T) {
	desc := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      8,
		Alignment: 4,
		Members: []*types.TypeDescriptor{
			types.UInt32TypeDescriptor,
			types.FloatTypeDescriptor,
		},
	}

	ints, floats := countStructRegUsage(desc)
	if ints != 1 || floats != 0 {
		t.Fatalf("countStructRegUsage = (%d,%d), want (1,0)", ints, floats)
	}
}

func TestCountStructRegUsageHFA(t *testing.T) {
	desc := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      32,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			types.DoubleTypeDescriptor,
			types.DoubleTypeDescriptor,
			types.DoubleTypeDescriptor,
			types.DoubleTypeDescriptor,
		},
	}

	ints, floats := countStructRegUsage(desc)
	if ints != 0 || floats != 4 {
		t.Fatalf("countStructRegUsage = (%d,%d), want (0,4)", ints, floats)
	}
}

func TestPlaceStructRegistersConcurrent(t *testing.T) {
	type NSSize struct {
		Width  float64
		Height float64
	}

	desc := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      16,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			types.DoubleTypeDescriptor,
			types.DoubleTypeDescriptor,
		},
	}

	val := NSSize{Width: 123.0, Height: 456.0}
	want0 := math.Float64bits(val.Width)
	want1 := math.Float64bits(val.Height)

	const workers = 64
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		go func() {
			_, fpr, gprIdx, fprIdx := placeStruct(t, desc, unsafe.Pointer(&val))
			if gprIdx != 0 {
				errCh <- fmt.Errorf("gprIdx = %d", gprIdx)
				return
			}
			if fprIdx != 2 {
				errCh <- fmt.Errorf("fprIdx = %d", fprIdx)
				return
			}
			if fpr[0] != want0 || fpr[1] != want1 {
				errCh <- fmt.Errorf("fpr = [0x%x 0x%x]", fpr[0], fpr[1])
				return
			}
			errCh <- nil
		}()
	}

	for i := 0; i < workers; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("concurrent pack failed: %v", err)
		}
	}
}
