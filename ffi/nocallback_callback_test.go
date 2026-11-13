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

// TestCallFunctionNoCallbackRejectsCallback checks that a C function that
// calls back into Go during CallFunctionNoCallback stops the process with the
// runtime's message instead of running Go code on a stack the caller's
// values may live on.
func TestCallFunctionNoCallbackRejectsCallback(t *testing.T) {
	if os.Getenv("GOFFI_NOCALLBACK_CHILD") != "1" {
		requireStructLib(t)
		exe, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(exe, "-test.run=^TestCallFunctionNoCallbackRejectsCallback$", "-test.count=1")
		cmd.Env = append(os.Environ(), "GOFFI_NOCALLBACK_CHILD=1", structTestLibEnv+"="+structTestLibPath)
		out, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(out), "called back into Go") {
			t.Fatalf("child process: %v\n%s", err, out)
		}
		return
	}

	cif, fn := prepareTestFunc(t, "cb_call_double_ret", types.DoubleTypeDescriptor,
		types.PointerTypeDescriptor, types.SInt64TypeDescriptor)
	cb := NewCallback(func(a int64) float64 { return float64(a) })
	a := int64(3)
	var r float64
	CallFunctionNoCallback(cif, fn, unsafe.Pointer(&r), []unsafe.Pointer{unsafe.Pointer(&cb), unsafe.Pointer(&a)})
	t.Fatal("the callback ran")
}
