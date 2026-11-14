// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

package ffi

import (
	"errors"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// prepareAbs prepares the system C library's int abs(int), so these tests run
// where the C test library is not built.
func prepareAbs(t *testing.T) (*types.CallInterface, unsafe.Pointer) {
	t.Helper()
	lib, err := LoadLibrary(systemLibC(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { FreeLibrary(lib) })
	fn, err := GetSymbol(lib, "abs")
	if err != nil {
		t.Fatal(err)
	}
	cif := new(types.CallInterface)
	if err := PrepareCallInterface(cif, systemConvention(), types.SInt32TypeDescriptor,
		[]*types.TypeDescriptor{types.SInt32TypeDescriptor}); err != nil {
		t.Fatal(err)
	}
	return cif, fn
}

// TestCallFunctionNoCallbackAllocs checks that a call through a bridge
// allocates nothing: the value, avalue and the result stay on the stack.
func TestCallFunctionNoCallbackAllocs(t *testing.T) {
	cif, fn := prepareAbs(t)
	if cif.Bridge == nil {
		t.Skip("no call bridge on this platform")
	}
	var r int32
	allocs := testing.AllocsPerRun(100, func() {
		a := int32(-7)
		if _, err := CallFunctionNoCallback(cif, fn, unsafe.Pointer(&r),
			[]unsafe.Pointer{unsafe.Pointer(&a)}); err != nil {
			t.Fatal(err)
		}
	})
	if allocs != 0 {
		t.Fatalf("%v allocations per call, want 0", allocs)
	}
	if r != 7 {
		t.Fatalf("abs(-7) = %d, want 7", r)
	}
}

// TestCallFunctionNoCallbackInvalidCall checks the errors returned before C is
// called, and that a valid call works after them.
func TestCallFunctionNoCallbackInvalidCall(t *testing.T) {
	cif, fn := prepareAbs(t)
	// The checks reject the call before fn runs, so this signature need not
	// be abs's.
	two := new(types.CallInterface)
	if err := PrepareCallInterface(two, systemConvention(), types.SInt32TypeDescriptor,
		[]*types.TypeDescriptor{types.SInt32TypeDescriptor, types.SInt32TypeDescriptor}); err != nil {
		t.Fatal(err)
	}
	a := int32(-3)
	var r int32
	rvalue, avalue := unsafe.Pointer(&r), []unsafe.Pointer{unsafe.Pointer(&a)}
	tests := []struct {
		name   string
		cif    *types.CallInterface
		fn     unsafe.Pointer
		avalue []unsafe.Pointer
		field  string
		index  int
	}{
		{"nil cif", nil, fn, avalue, "cif", -1},
		{"nil fn", cif, nil, avalue, "fn", -1},
		{"missing argument", cif, fn, nil, "avalue", -1},
		{"nil argument", two, fn, []unsafe.Pointer{unsafe.Pointer(&a), nil}, "avalue", 1},
	}
	for _, tt := range tests {
		_, err := CallFunctionNoCallback(tt.cif, tt.fn, rvalue, tt.avalue)
		var invalid *InvalidCallInterfaceError
		if !errors.As(err, &invalid) || invalid.Field != tt.field || invalid.Index != tt.index {
			t.Errorf("%s: err = %v, want an InvalidCallInterfaceError for %s at index %d",
				tt.name, err, tt.field, tt.index)
		}
	}
	if _, err := CallFunctionNoCallback(cif, fn, rvalue, avalue); err != nil || r != 3 {
		t.Fatalf("abs(-3) after the rejected calls = %d, %v; want 3", r, err)
	}
}
