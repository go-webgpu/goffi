// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

package ffi

import (
	"testing"

	"github.com/go-webgpu/goffi/internal/arch"
	"github.com/go-webgpu/goffi/types"
)

// bridgeCompiler returns the platform's bridge compiler, or skips the test
// when there is none or bridges are disabled (GOFFI_JIT=0).
func bridgeCompiler(t *testing.T) arch.Compiler {
	t.Helper()
	c, ok := arch.Registry.Caller.(arch.Compiler)
	if !ok || c.Compile(&types.CallInterface{ReturnType: types.VoidTypeDescriptor}) == nil {
		t.Skip("no call bridges on this platform")
	}
	return c
}

func TestPrepareBridge(t *testing.T) {
	bridgeCompiler(t)
	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.SInt32TypeDescriptor,
		[]*types.TypeDescriptor{types.PointerTypeDescriptor, types.DoubleTypeDescriptor}); err != nil {
		t.Fatal(err)
	}
	if cif.Bridge == nil {
		t.Fatal("PrepareCallInterface compiled no bridge for a scalar signature")
	}
	pair := &types.TypeDescriptor{Kind: types.StructType,
		Members: []*types.TypeDescriptor{types.SInt32TypeDescriptor, types.SInt32TypeDescriptor}}
	if err := PrepareCallInterface(&cif, types.DefaultCall, pair, nil); err != nil {
		t.Fatal(err)
	}
	if cif.Bridge != nil {
		t.Fatal("preparing the CallInterface again for a struct result kept the scalar bridge")
	}
}

// TestPrepareVariadicBridge checks that the bridge of a variadic call is
// compiled for its variadic layout (Apple arm64 passes variadic arguments
// on the stack).
func TestPrepareVariadicBridge(t *testing.T) {
	c := bridgeCompiler(t)
	var cif types.CallInterface
	if err := PrepareVariadicCallInterface(&cif, types.DefaultCall, 1, types.SInt64TypeDescriptor,
		[]*types.TypeDescriptor{types.SInt64TypeDescriptor, types.SInt32TypeDescriptor, types.DoubleTypeDescriptor}); err != nil {
		t.Fatal(err)
	}
	want := cif
	want.Bridge = nil
	if cif.Bridge != c.Compile(&want) {
		t.Fatal("the bridge was not compiled for the variadic layout")
	}
}
