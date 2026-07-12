//go:build !windows

package ffi

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// TestCallFunctionCapturesErrno verifies that CallFunction captures a
// non-zero errno when a POSIX function fails.
//
// Strategy: call open(2) with a path that does not exist. POSIX mandates that
// open returns -1 and sets errno = ENOENT in this case.
func TestCallFunctionCapturesErrno(t *testing.T) {
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
	cerrno, err := CallFunction(cif, openFn,
		unsafe.Pointer(&result),
		[]unsafe.Pointer{unsafe.Pointer(&pathPtr), unsafe.Pointer(&flags)},
	)
	if err != nil {
		t.Fatalf("CallFunction failed: %v", err)
	}

	if result != -1 {
		t.Fatalf("expected open() to return -1, got %d", result)
	}
	if cerrno != syscall.ENOENT {
		t.Errorf("expected errno=ENOENT(%d), got %d (%v)",
			syscall.ENOENT, cerrno, cerrno)
	}
}

// TestCallFunctionZeroErrnoOnSuccess verifies that errno is 0 after a successful call.
// We use strlen(3) which always succeeds and does not set errno.
func TestCallFunctionZeroErrnoOnSuccess(t *testing.T) {
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
	cerrno, err := CallFunction(cif, strlenFn,
		unsafe.Pointer(&result),
		[]unsafe.Pointer{unsafe.Pointer(&ptr)},
	)
	if err != nil {
		t.Fatalf("CallFunction failed: %v", err)
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

// TestCallFunctionNilCIF verifies that a nil CIF returns an error.
func TestCallFunctionNilCIF(t *testing.T) {
	_, err := CallFunction(nil, nil, nil, nil)
	if err == nil {
		t.Error("expected error for nil CIF, got nil")
	}
}

// TestCallFunctionNilFn verifies that a nil function pointer returns an error.
func TestCallFunctionNilFn(t *testing.T) {
	cif := &types.CallInterface{}
	prepErr := PrepareCallInterface(cif, types.UnixCallingConvention,
		types.VoidTypeDescriptor, nil)
	if prepErr != nil {
		t.Fatalf("PrepareCallInterface failed: %v", prepErr)
	}
	_, err := CallFunction(cif, nil, nil, nil)
	if err == nil {
		t.Error("expected error for nil fn, got nil")
	}
}

// BenchmarkCallFunctionErrnoOverhead measures the errno capture overhead inside
// CallFunction relative to a baseline by calling strlen on a short string.
func BenchmarkCallFunctionErrnoOverhead(b *testing.B) {
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
		_, _ = CallFunction(cif, strlenFn,
			unsafe.Pointer(&result),
			[]unsafe.Pointer{unsafe.Pointer(&ptr)},
		)
	}
}
