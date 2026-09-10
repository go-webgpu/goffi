//go:build goffi_static

package ffi_test

import (
	"errors"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/ffi"
)

func TestStaticBuild_LoadLibraryReturnsErrStaticBuild(t *testing.T) {
	handle, err := ffi.LoadLibrary("libc.so.6")
	if handle != nil {
		t.Fatalf("LoadLibrary handle = %v, want nil", handle)
	}
	if !errors.Is(err, ffi.ErrStaticBuild) {
		t.Fatalf("LoadLibrary error = %v, want errors.Is(..., ErrStaticBuild)", err)
	}

	var libErr *ffi.LibraryError
	if !errors.As(err, &libErr) {
		t.Fatalf("LoadLibrary error type = %T, want *LibraryError", err)
	}
	if libErr.Operation != "load" {
		t.Fatalf("Operation = %q, want load", libErr.Operation)
	}
}

func TestStaticBuild_GetSymbolReturnsErrStaticBuild(t *testing.T) {
	_, err := ffi.GetSymbol(unsafe.Pointer(uintptr(1)), "strlen")
	if !errors.Is(err, ffi.ErrStaticBuild) {
		t.Fatalf("GetSymbol error = %v, want errors.Is(..., ErrStaticBuild)", err)
	}
}

func TestStaticBuild_FreeLibraryNilOK(t *testing.T) {
	if err := ffi.FreeLibrary(nil); err != nil {
		t.Fatalf("FreeLibrary(nil) = %v, want nil", err)
	}
}
