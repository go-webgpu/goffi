//go:build (linux || darwin || freebsd) && (amd64 || arm64)

package ffi

import (
	"runtime"
	"sync/atomic"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// TestCallback_FromCThread verifies that a goffi callback fires correctly
// when invoked from an OS thread the Go runtime has never seen, i.e. one
// created by C via pthread_create. The interesting path is:
//
//	C-created thread -> trampoline (callback_{amd64,arm64}.s)
//	  -> crosscall2 -> runtime.cgocallback -> Go closure
//
// Under CGO_ENABLED=0, crosscall2 is provided by internal/fakecgo.
// Under CGO_ENABLED=1, crosscall2 is provided by runtime/cgo (and is
// byte-identical to fakecgo's copy on the supported targets). The
// assertion below — that the Go closure ran with the expected arg and
// that pthread_join observed the closure's return value — is identical
// in both modes, so running this test in CI under both cgo=0 and cgo=1
// is the explicit confirmation requested in PR #37 that the callback
// path through runtime/cgo's crosscall2 is exercised end-to-end.
func TestCallback_FromCThread(t *testing.T) {
	var libCandidates []string
	switch runtime.GOOS {
	case "linux":
		// glibc >= 2.34 ships pthread_create in libc.so.6 (libpthread.so.0
		// is a stub mapping to libc.so.6). Older glibc and musl still
		// ship pthread_create in libpthread.so.0. Try both.
		libCandidates = []string{"libpthread.so.0", "libc.so.6"}
	case "darwin":
		libCandidates = []string{"/usr/lib/libSystem.B.dylib", "libSystem.B.dylib"}
	case "freebsd":
		libCandidates = []string{"libthr.so.3"}
	default:
		t.Skipf("pthread_create not exercised on %s", runtime.GOOS)
	}

	var (
		handle  unsafe.Pointer
		create  unsafe.Pointer
		join    unsafe.Pointer
		lastErr error
	)
	for _, name := range libCandidates {
		h, err := LoadLibrary(name)
		if err != nil {
			lastErr = err
			continue
		}
		cre, errCre := GetSymbol(h, "pthread_create")
		joi, errJoi := GetSymbol(h, "pthread_join")
		if errCre != nil || errJoi != nil {
			_ = FreeLibrary(h)
			if errCre != nil {
				lastErr = errCre
			} else {
				lastErr = errJoi
			}
			continue
		}
		handle, create, join = h, cre, joi
		break
	}
	if handle == nil {
		t.Skipf("pthread_create / pthread_join not found in any candidate library: %v", lastErr)
	}
	defer func() { _ = FreeLibrary(handle) }()

	// int pthread_create(pthread_t *tid, const pthread_attr_t *attr,
	//                    void *(*start)(void *), void *arg)
	cifCreate := &types.CallInterface{}
	if err := PrepareCallInterface(cifCreate, types.UnixCallingConvention,
		types.SInt32TypeDescriptor,
		[]*types.TypeDescriptor{
			types.PointerTypeDescriptor,
			types.PointerTypeDescriptor,
			types.PointerTypeDescriptor,
			types.PointerTypeDescriptor,
		}); err != nil {
		t.Fatalf("PrepareCallInterface(pthread_create): %v", err)
	}

	// int pthread_join(pthread_t tid, void **retval)
	// pthread_t is uintptr-sized on every (linux|darwin|freebsd) target
	// goffi supports today (glibc unsigned long, darwin opaque pointer,
	// freebsd opaque pointer). We pass it through PointerTypeDescriptor
	// for that reason; the ABI classification is identical to UInt64.
	cifJoin := &types.CallInterface{}
	if err := PrepareCallInterface(cifJoin, types.UnixCallingConvention,
		types.SInt32TypeDescriptor,
		[]*types.TypeDescriptor{
			types.PointerTypeDescriptor,
			types.PointerTypeDescriptor,
		}); err != nil {
		t.Fatalf("PrepareCallInterface(pthread_join): %v", err)
	}

	// The callback runs on a brand new C thread. If the FFI did not
	// transition to a fresh goroutine first, the Go runtime would crash
	// long before reaching this Go code. The arg/return values are
	// our end-to-end witnesses.
	var observedArg atomic.Uint64
	var observedG atomic.Uintptr
	cb := NewCallback(func(arg uintptr) uintptr {
		observedArg.Store(uint64(arg))
		// Record a value that proves we are inside Go (a non-zero
		// goroutine identity sourced from a Go-only construct: the
		// address of a stack variable in this closure).
		var sentinel int
		observedG.Store(uintptr(unsafe.Pointer(&sentinel)))
		return 0xCAFEBABE
	})
	if cb == 0 {
		t.Fatal("NewCallback returned 0")
	}

	const argSentinel uintptr = 0xC0FFEE

	// pthread_create call.
	var tid uintptr
	tidAddr := unsafe.Pointer(&tid) // must outlive the call; keeps tid pinned
	var attrPtr unsafe.Pointer      // NULL attr
	cbVal := cb                     // start_routine
	argVal := argSentinel           // arg
	avalueCreate := []unsafe.Pointer{
		unsafe.Pointer(&tidAddr),
		unsafe.Pointer(&attrPtr),
		unsafe.Pointer(&cbVal),
		unsafe.Pointer(&argVal),
	}
	var rcCreate int32
	if err := CallFunction(cifCreate, create, unsafe.Pointer(&rcCreate), avalueCreate); err != nil {
		t.Fatalf("CallFunction(pthread_create): %v", err)
	}
	if rcCreate != 0 {
		t.Fatalf("pthread_create returned %d", rcCreate)
	}

	// pthread_join — blocks until the callback returns, and gives us
	// the callback's return value via *retvalSlot.
	var retvalSlot uintptr
	retvalAddr := unsafe.Pointer(&retvalSlot)
	tidVal := tid
	avalueJoin := []unsafe.Pointer{
		unsafe.Pointer(&tidVal),
		unsafe.Pointer(&retvalAddr),
	}
	var rcJoin int32
	if err := CallFunction(cifJoin, join, unsafe.Pointer(&rcJoin), avalueJoin); err != nil {
		t.Fatalf("CallFunction(pthread_join): %v", err)
	}
	if rcJoin != 0 {
		t.Fatalf("pthread_join returned %d", rcJoin)
	}

	if got := observedArg.Load(); got != uint64(argSentinel) {
		t.Errorf("callback observed arg = %#x, want %#x", got, argSentinel)
	}
	if observedG.Load() == 0 {
		t.Errorf("callback did not appear to run inside a Go context (no stack address recorded)")
	}
	if retvalSlot != 0xCAFEBABE {
		t.Errorf("pthread_join retval = %#x, want %#x", retvalSlot, uintptr(0xCAFEBABE))
	}
}
