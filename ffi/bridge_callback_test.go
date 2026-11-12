// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build ((linux && !android) || darwin || freebsd) && (amd64 || arm64)

package ffi

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// TestBridgeCallbackMovesStack makes a call through a bridge whose C
// function calls back into Go, and the callback grows the goroutine stack,
// which moves it. The bridge's frame lives on that stack, so the bridge must
// not touch it after the C call. GODEBUG=efence=1 makes freed stacks fault,
// so a bridge that did would crash the child process.
func TestBridgeCallbackMovesStack(t *testing.T) {
	if os.Getenv("GOFFI_EFENCE_CHILD") != "1" {
		requireStructLib(t)
		exe, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(exe, "-test.run=^TestBridgeCallbackMovesStack$", "-test.count=1", "-test.v")
		cmd.Env = append(os.Environ(), "GOFFI_EFENCE_CHILD=1", "GODEBUG=efence=1",
			structTestLibEnv+"="+structTestLibPath)
		out, err := cmd.CombinedOutput()
		if err != nil || !strings.Contains(string(out), "--- PASS: TestBridgeCallbackMovesStack") {
			t.Fatalf("child process: %v\n%s", err, out)
		}
		return
	}

	moved := false
	cb := NewCallback(func(a int64) float64 {
		before := uintptr(unsafe.Pointer(&a))
		v := grow(2000, a)
		moved = uintptr(unsafe.Pointer(&a)) != before
		return float64(v) / 2
	})
	a := int64(1234)
	var result float64
	callCFunc(t, "cb_call_double_ret", types.DoubleTypeDescriptor, unsafe.Pointer(&result),
		[]*types.TypeDescriptor{types.PointerTypeDescriptor, types.SInt64TypeDescriptor},
		unsafe.Pointer(&cb), unsafe.Pointer(&a))
	if !moved {
		t.Fatal("the callback did not move the goroutine stack")
	}
	if result != 617 {
		t.Fatalf("result %v, want 617", result)
	}
}

// grow uses depth frames of about 600 bytes and returns v.
//
//go:noinline
func grow(depth int, v int64) int64 {
	var pad [512]byte
	pad[depth%len(pad)] = byte(depth)
	if depth == 0 {
		return v
	}
	return grow(depth-1, v) + int64(pad[depth%len(pad)]) - int64(byte(depth))
}
