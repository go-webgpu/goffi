// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !goffi_static

package ffi

import (
	"errors"
	"syscall"
	"testing"
)

// TestNamesWithNULAreRejected checks that a NUL byte in a library or symbol
// name is reported as an error on every OS. Before, Windows GetSymbol
// panicked (syscall.StringBytePtr) and the dlopen/dlsym paths silently cut
// the name at the NUL, so a different library or symbol could be used.
func TestNamesWithNULAreRejected(t *testing.T) {
	_, err := LoadLibrary("libgoffi-test\x00.so")
	checkNULNameError(t, err, "load")

	// The name is checked before the handle is used, so nil is fine here.
	_, err = GetSymbol(nil, "goffi_test\x00symbol")
	checkNULNameError(t, err, "symbol")
}

func checkNULNameError(t *testing.T, err error, operation string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected an error for a name containing NUL", operation)
	}
	var libErr *LibraryError
	if !errors.As(err, &libErr) {
		t.Fatalf("%s: expected *LibraryError, got %T: %v", operation, err, err)
	}
	if libErr.Operation != operation {
		t.Errorf("Operation = %q, want %q", libErr.Operation, operation)
	}
	if !errors.Is(err, syscall.EINVAL) {
		t.Errorf("%s: expected syscall.EINVAL, got %v", operation, libErr.Err)
	}
}
