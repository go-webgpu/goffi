// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build linux && !android && !goffi_static

package ffi

import (
	"errors"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/internal/hostlibc"
	"github.com/go-webgpu/goffi/types"
)

// TestNoHostLibcIsReportedNotFatal covers the state a universal binary lands
// in on a host with no dynamic loader goffi recognises: startup could not bind
// a libc, so every imported symbol is unbound and calling one would fault.
//
// The bridge sets hostlibc.Missing there, before the runtime starts, and the
// point of the flag is that the process then behaves like a build without FFI
// rather than crashing -- Available says so up front, and the three entry
// points that need libc return ErrNoHostLibc. Reaching that state for real
// needs a machine with no ld.so, so the flag is set directly here; ffi's tests
// run sequentially, so no concurrent call observes it.
func TestNoHostLibcIsReportedNotFatal(t *testing.T) {
	saved := hostlibc.Missing
	hostlibc.Missing = true
	t.Cleanup(func() { hostlibc.Missing = saved })

	if Available() {
		t.Error("Available() = true with no host libc, want false")
	}

	if _, err := LoadLibrary("libc.so.6"); !errors.Is(err, ErrNoHostLibc) {
		t.Errorf("LoadLibrary error = %v, want one wrapping ErrNoHostLibc", err)
	}

	if _, err := GetSymbol(nil, "strlen"); !errors.Is(err, ErrNoHostLibc) {
		t.Errorf("GetSymbol error = %v, want one wrapping ErrNoHostLibc", err)
	}

	// executeFunction is the guard CallFunction goes through. A caller cannot
	// legitimately hold a foreign function pointer in this mode -- the two
	// calls above are the only ways to get one, and both fail -- so this is
	// belt and braces, checked with a pointer that must never be dereferenced.
	var cif types.CallInterface
	if _, err := executeFunction(&cif, unsafe.Pointer(uintptr(1)), nil, nil); !errors.Is(err, ErrNoHostLibc) {
		t.Errorf("executeFunction error = %v, want ErrNoHostLibc", err)
	}
}

// TestHostLibcPresentByDefault guards the flag's default: every build that is
// not the universal one, and every universal build on a host goffi can bind a
// libc on, must leave it false. A stray set would silently disable FFI
// everywhere.
func TestHostLibcPresentByDefault(t *testing.T) {
	if hostlibc.Missing {
		t.Fatal("hostlibc.Missing is set on a host that has a libc")
	}
	if !Available() {
		t.Error("Available() = false in a non-static build with a libc")
	}
}
