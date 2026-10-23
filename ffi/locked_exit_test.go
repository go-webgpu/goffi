//go:build !windows

package ffi

import (
	"runtime"
	"sync"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// TestCallThenLockedExit verifies that a goroutine can make a C call and then
// exit while it still holds runtime.LockOSThread.
//
// The runtime terminates the thread of such a goroutine, so mstart0 returns
// through its frame on the g0 stack. A call trampoline that writes outside its
// own frame on the g0 stack can corrupt that return path and crash the process
// with SIGSEGV when the thread exits.
func TestCallThenLockedExit(t *testing.T) {
	var libName string
	switch runtime.GOOS {
	case "linux":
		libName = "libc.so.6"
	case "darwin":
		libName = "libSystem.B.dylib"
	case "freebsd":
		libName = "libc.so.7"
	default:
		t.Skipf("locked exit after a call not tested on %s", runtime.GOOS)
	}

	handle, err := LoadLibrary(libName)
	if err != nil {
		t.Fatalf("LoadLibrary(%s) failed: %v", libName, err)
	}
	defer FreeLibrary(handle)

	getpid, err := GetSymbol(handle, "getpid")
	if err != nil {
		t.Fatalf("GetSymbol(getpid) failed: %v", err)
	}

	cif := &types.CallInterface{}
	err = PrepareCallInterface(cif, types.UnixCallingConvention, types.SInt32TypeDescriptor, nil)
	if err != nil {
		t.Fatalf("PrepareCallInterface failed: %v", err)
	}

	const rounds = 10
	const workers = 16
	for range rounds {
		var wg sync.WaitGroup
		wg.Add(workers)
		for range workers {
			go func() {
				defer wg.Done()
				runtime.LockOSThread()
				var pid int32
				if _, err := CallFunction(cif, getpid, unsafe.Pointer(&pid), nil); err != nil {
					t.Errorf("CallFunction(getpid) failed: %v", err)
				}
				// Intentionally no UnlockOSThread: the goroutine exits with its
				// thread locked, so the M exits and mstart returns.
			}()
		}
		wg.Wait()
	}
}
