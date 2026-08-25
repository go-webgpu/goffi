//go:build linux && !android && cgo

package cbridge

import "testing"

const rtldNow = 0x00002

func openStandardCLibrary(t *testing.T) uintptr {
	t.Helper()

	candidates := []string{
		"libc.so.6",
		"libdl.so.2",
		"libc.musl-x86_64.so.1",
		"libc.musl-aarch64.so.1",
	}
	for _, candidate := range candidates {
		if handle, _ := Dlopen(candidate, rtldNow); handle != 0 {
			return handle
		}
	}

	t.Skipf("no standard libc or libdl candidate available; tried %v", candidates)
	return 0
}

func TestDlopenAndDlsym(t *testing.T) {
	handle := openStandardCLibrary(t)

	symbol, errMessage := Dlsym(handle, "malloc")
	if symbol == 0 {
		t.Fatalf("Dlsym(malloc) returned 0: %s", errMessage)
	}
	if errMessage != "" {
		t.Fatalf("Dlsym(malloc) returned an unexpected error: %s", errMessage)
	}
}

func TestDlopenInvalidLibrary(t *testing.T) {
	handle, errMessage := Dlopen("libgoffi-does-not-exist.so", rtldNow)
	if handle != 0 {
		t.Fatalf("Dlopen returned handle %#x for an invalid library", handle)
	}
	if errMessage == "" {
		t.Fatal("Dlopen returned an empty error for an invalid library")
	}
}

func TestErrnoLocationAddress(t *testing.T) {
	if address := ErrnoLocationAddress(); address == 0 {
		t.Fatal("ErrnoLocationAddress returned 0")
	}
}
