//go:build !goffi_static

package ffi

import (
	"errors"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/internal/arch"
	"github.com/go-webgpu/goffi/types"
)

// TestCallFunctionUnsupportedArchitecture covers executeFunction's guard that
// rejects a call when no architecture caller is registered, which is the state
// on a GOARCH goffi does not implement. arch.Registry is a process-global, so
// the caller is swapped out and restored within this one test; ffi tests run
// sequentially (none call t.Parallel), so no concurrent call observes the nil
// caller. A non-nil cif and fn are required only to pass CallFunctionContext's
// argument validation; the guard returns before either is dereferenced.
func TestCallFunctionUnsupportedArchitecture(t *testing.T) {
	saved := arch.Registry.Caller
	arch.Registry.Caller = nil
	t.Cleanup(func() { arch.Registry.Caller = saved })

	var dummy int
	cif := &types.CallInterface{}
	_, err := CallFunction(cif, unsafe.Pointer(&dummy), nil, nil)
	if !errors.Is(err, types.ErrUnsupportedArchitecture) {
		t.Fatalf("CallFunction with no registered caller: got %v, want ErrUnsupportedArchitecture", err)
	}
}
