//go:build !windows

package ffi

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// TestCallFunctionErrnoBasic verifies that CallFunctionErrno captures a
// non-zero errno when a POSIX function fails.
//
// Strategy: call open(2) with a path that does not exist. POSIX mandates that
// open returns -1 and sets errno = ENOENT in this case.
func TestCallFunctionErrnoBasic(t *testing.T) {
	var libName string
	switch runtime.GOOS {
	case "linux":
		libName = "libc.so.6"
	case "darwin":
		libName = "libSystem.B.dylib"
	case "freebsd":
		libName = "libc.so.7"
	default:
		t.Skipf("errno capture not tested on %s", runtime.GOOS)
	}

	handle, err := LoadLibrary(libName)
	if err != nil {
		t.Fatalf("LoadLibrary(%s) failed: %v", libName, err)
	}
	defer FreeLibrary(handle)

	openFn, err := GetSymbol(handle, "open")
	if err != nil {
		t.Fatalf("GetSymbol(open) failed: %v", err)
	}

	// Prepare CIF: int open(const char *pathname, int flags)
	cif := &types.CallInterface{}
	err = PrepareCallInterface(cif, types.UnixCallingConvention,
		types.SInt32TypeDescriptor,
		[]*types.TypeDescriptor{types.PointerTypeDescriptor, types.SInt32TypeDescriptor},
	)
	if err != nil {
		t.Fatalf("PrepareCallInterface failed: %v", err)
	}

	// Call open("/goffi_test_nonexistent_path_abc\x00", 0) — must return -1 / ENOENT.
	path := "/goffi_test_nonexistent_path_abc\x00"
	pathPtr := unsafe.Pointer(unsafe.StringData(path))
	flags := int32(0) // O_RDONLY

	var result int32
	cerrno, err := CallFunctionErrno(cif, openFn,
		unsafe.Pointer(&result),
		[]unsafe.Pointer{unsafe.Pointer(&pathPtr), unsafe.Pointer(&flags)},
	)
	if err != nil {
		t.Fatalf("CallFunctionErrno failed: %v", err)
	}

	if result != -1 {
		t.Fatalf("expected open() to return -1, got %d", result)
	}
	if cerrno != uintptr(syscall.ENOENT) {
		t.Errorf("expected errno=ENOENT(%d), got %d (%v)",
			syscall.ENOENT, cerrno, syscall.Errno(cerrno))
	}
}

// TestCallFunctionErrnoSuccess verifies that errno is 0 after a successful call.
// We use strlen(3) which always succeeds and does not set errno.
func TestCallFunctionErrnoSuccess(t *testing.T) {
	var libName string
	switch runtime.GOOS {
	case "linux":
		libName = "libc.so.6"
	case "darwin":
		libName = "libSystem.B.dylib"
	case "freebsd":
		libName = "libc.so.7"
	default:
		t.Skipf("errno capture not tested on %s", runtime.GOOS)
	}

	handle, err := LoadLibrary(libName)
	if err != nil {
		t.Fatalf("LoadLibrary(%s) failed: %v", libName, err)
	}
	defer FreeLibrary(handle)

	strlenFn, err := GetSymbol(handle, "strlen")
	if err != nil {
		t.Fatalf("GetSymbol(strlen) failed: %v", err)
	}

	cif := &types.CallInterface{}
	err = PrepareCallInterface(cif, types.UnixCallingConvention,
		types.UInt64TypeDescriptor,
		[]*types.TypeDescriptor{types.PointerTypeDescriptor},
	)
	if err != nil {
		t.Fatalf("PrepareCallInterface failed: %v", err)
	}

	input := "hello\x00"
	ptr := unsafe.Pointer(unsafe.StringData(input))

	var result uint64
	cerrno, err := CallFunctionErrno(cif, strlenFn,
		unsafe.Pointer(&result),
		[]unsafe.Pointer{unsafe.Pointer(&ptr)},
	)
	if err != nil {
		t.Fatalf("CallFunctionErrno failed: %v", err)
	}
	if result != 5 {
		t.Errorf("strlen returned %d, want 5", result)
	}
	// errno is not guaranteed to be 0 after a successful call (per POSIX),
	// but for strlen it should be. We log rather than fail hard.
	if cerrno != 0 {
		t.Logf("note: errno=%d after successful strlen (unexpected but not fatal)", cerrno)
	}
}

// TestCallFunctionErrnoNilCIF verifies that a nil CIF returns an error.
func TestCallFunctionErrnoNilCIF(t *testing.T) {
	_, err := CallFunctionErrno(nil, nil, nil, nil)
	if err == nil {
		t.Error("expected error for nil CIF, got nil")
	}
}

// TestCallFunctionErrnoNilFn verifies that a nil function pointer returns an error.
func TestCallFunctionErrnoNilFn(t *testing.T) {
	cif := &types.CallInterface{}
	prepErr := PrepareCallInterface(cif, types.UnixCallingConvention,
		types.VoidTypeDescriptor, nil)
	if prepErr != nil {
		t.Fatalf("PrepareCallInterface failed: %v", prepErr)
	}
	_, err := CallFunctionErrno(cif, nil, nil, nil)
	if err == nil {
		t.Error("expected error for nil fn, got nil")
	}
}

// BenchmarkCallFunctionErrno measures the overhead of CallFunctionErrno
// relative to CallFunction by calling strlen on a short string.
func BenchmarkCallFunctionErrno(b *testing.B) {
	var libName string
	switch runtime.GOOS {
	case "linux":
		libName = "libc.so.6"
	case "darwin":
		libName = "libSystem.B.dylib"
	case "freebsd":
		libName = "libc.so.7"
	default:
		b.Skipf("errno capture not benchmarked on %s", runtime.GOOS)
	}

	handle, err := LoadLibrary(libName)
	if err != nil {
		b.Fatalf("LoadLibrary failed: %v", err)
	}
	defer FreeLibrary(handle)

	strlenFn, err := GetSymbol(handle, "strlen")
	if err != nil {
		b.Fatalf("GetSymbol(strlen) failed: %v", err)
	}

	cif := &types.CallInterface{}
	if err = PrepareCallInterface(cif, types.UnixCallingConvention,
		types.UInt64TypeDescriptor,
		[]*types.TypeDescriptor{types.PointerTypeDescriptor},
	); err != nil {
		b.Fatalf("PrepareCallInterface failed: %v", err)
	}

	input := "benchmark\x00"
	ptr := unsafe.Pointer(unsafe.StringData(input))

	var result uint64
	b.ResetTimer()
	for b.Loop() {
		_, _ = CallFunctionErrno(cif, strlenFn,
			unsafe.Pointer(&result),
			[]unsafe.Pointer{unsafe.Pointer(&ptr)},
		)
	}
}
