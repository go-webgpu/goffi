// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build windows && amd64

package ffi

import (
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

func TestWindowsAMD64ScalarFloatReturns(t *testing.T) {
	requireStructLib(t)

	token := byte(1)
	pointerArg := unsafe.Pointer(&token)
	arguments := []unsafe.Pointer{unsafe.Pointer(&pointerArg)}

	t.Run("float32", func(t *testing.T) {
		symbol := requireTestSymbol(t, "return_float32")
		cif := prepareWindowsFloatCall(t, types.FloatTypeDescriptor)

		var result float32
		if _, err := CallFunction(cif, symbol, unsafe.Pointer(&result), arguments); err != nil {
			t.Fatal(err)
		}
		if result != 0.125 {
			t.Fatalf("return_float32() = %v, want 0.125", result)
		}
	})

	t.Run("float64", func(t *testing.T) {
		symbol := requireTestSymbol(t, "return_float64")
		cif := prepareWindowsFloatCall(t, types.DoubleTypeDescriptor)

		var result float64
		if _, err := CallFunction(cif, symbol, unsafe.Pointer(&result), arguments); err != nil {
			t.Fatal(err)
		}
		if result != 0.625 {
			t.Fatalf("return_float64() = %v, want 0.625", result)
		}
	})
}

func requireTestSymbol(t *testing.T, name string) unsafe.Pointer {
	t.Helper()
	symbol, err := GetSymbol(structTestLib, name)
	if err != nil {
		t.Fatal(err)
	}
	return symbol
}

func prepareWindowsFloatCall(t *testing.T, returnType *types.TypeDescriptor) *types.CallInterface {
	t.Helper()
	cif := &types.CallInterface{}
	if err := PrepareCallInterface(
		cif,
		types.WindowsCallingConvention,
		returnType,
		[]*types.TypeDescriptor{types.PointerTypeDescriptor},
	); err != nil {
		t.Fatal(err)
	}
	return cif
}
