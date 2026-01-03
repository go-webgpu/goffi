package ffi

import (
	"context"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// TestFreeLibrary tests library cleanup on both platforms.
func TestFreeLibrary(t *testing.T) {
	var libName string
	switch runtime.GOOS {
	case "linux":
		libName = "libc.so.6"
	case "darwin":
		libName = "libSystem.B.dylib"
	case "windows":
		libName = "msvcrt.dll"
	default:
		t.Skip("Unsupported OS")
	}

	// Test successful free
	handle, err := LoadLibrary(libName)
	if err != nil {
		t.Fatalf("LoadLibrary failed: %v", err)
	}

	err = FreeLibrary(handle)
	if err != nil {
		t.Errorf("FreeLibrary failed: %v", err)
	}

	// Test nil handle (should not error)
	err = FreeLibrary(nil)
	if err != nil {
		t.Errorf("FreeLibrary(nil) should not error: %v", err)
	}
}

// TestCallFunctionContext tests context-aware function calls.
func TestCallFunctionContext(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		t.Skip("Test requires Linux, Windows, or macOS")
	}

	var libName, funcName string
	switch runtime.GOOS {
	case "linux":
		libName = "libc.so.6"
		funcName = "puts"
	case "darwin":
		libName = "libSystem.B.dylib"
		funcName = "puts"
	case "windows":
		libName = "msvcrt.dll"
		funcName = "printf"
	}

	handle, err := LoadLibrary(libName)
	if err != nil {
		t.Fatalf("LoadLibrary failed: %v", err)
	}
	defer FreeLibrary(handle)

	sym, err := GetSymbol(handle, funcName)
	if err != nil {
		t.Fatalf("GetSymbol failed: %v", err)
	}

	cif := &types.CallInterface{}
	err = PrepareCallInterface(cif, types.DefaultCall,
		types.SInt32TypeDescriptor,
		[]*types.TypeDescriptor{types.PointerTypeDescriptor})
	if err != nil {
		t.Fatalf("PrepareCallInterface failed: %v", err)
	}

	t.Run("SuccessfulCall", func(t *testing.T) {
		str := "Context test\n\x00"
		arg := unsafe.Pointer(unsafe.StringData(str))
		var retVal int32

		ctx := context.Background()
		// IMPORTANT: avalue contains pointers TO the argument values
		err := CallFunctionContext(ctx, cif, sym, unsafe.Pointer(&retVal), []unsafe.Pointer{unsafe.Pointer(&arg)})
		if err != nil {
			t.Errorf("CallFunctionContext failed: %v", err)
		}
	})

	t.Run("CancelledContext", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		str := "Should not print\n\x00"
		arg := unsafe.Pointer(unsafe.StringData(str))
		var retVal int32

		err := CallFunctionContext(ctx, cif, sym, unsafe.Pointer(&retVal), []unsafe.Pointer{unsafe.Pointer(&arg)})
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	})

	t.Run("TimeoutContext", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		time.Sleep(10 * time.Millisecond) // Ensure timeout before call
		defer cancel()

		str := "Should not print\n\x00"
		arg := unsafe.Pointer(unsafe.StringData(str))
		var retVal int32

		err := CallFunctionContext(ctx, cif, sym, unsafe.Pointer(&retVal), []unsafe.Pointer{unsafe.Pointer(&arg)})
		if err != context.DeadlineExceeded {
			t.Errorf("Expected context.DeadlineExceeded, got %v", err)
		}
	})

	t.Run("NilCIF", func(t *testing.T) {
		err := CallFunctionContext(context.Background(), nil, sym, nil, nil)
		var icErr *InvalidCallInterfaceError
		if err == nil || err.(*InvalidCallInterfaceError).Field != "cif" {
			t.Errorf("Expected InvalidCallInterfaceError for cif, got %v", err)
		}
		_ = icErr
	})

	t.Run("NilFunction", func(t *testing.T) {
		err := CallFunctionContext(context.Background(), cif, nil, nil, nil)
		if err == nil || err.(*InvalidCallInterfaceError).Field != "fn" {
			t.Errorf("Expected InvalidCallInterfaceError for fn, got %v", err)
		}
	})
}

// TestCompositeTypes tests struct type handling.
func TestCompositeTypes(t *testing.T) {
	t.Run("ValidStruct", func(t *testing.T) {
		// Create a simple struct type
		structType := &types.TypeDescriptor{
			Kind: types.StructType,
			Members: []*types.TypeDescriptor{
				types.SInt32TypeDescriptor,
				types.SInt32TypeDescriptor,
			},
		}

		cif := &types.CallInterface{}
		err := PrepareCallInterface(cif, types.DefaultCall, structType, nil)
		if err != nil {
			t.Errorf("PrepareCallInterface with struct failed: %v", err)
		}

		// Struct should now be initialized
		if structType.Size == 0 {
			t.Error("Struct size should be non-zero after initialization")
		}
	})

	t.Run("InvalidStructMember", func(t *testing.T) {
		// Struct with invalid member
		invalidMember := &types.TypeDescriptor{
			Kind:      types.TypeKind(999),
			Size:      4,
			Alignment: 4,
		}

		structType := &types.TypeDescriptor{
			Kind:    types.StructType,
			Members: []*types.TypeDescriptor{invalidMember},
		}

		cif := &types.CallInterface{}
		err := PrepareCallInterface(cif, types.DefaultCall, structType, nil)
		if err == nil {
			t.Error("Expected error for struct with invalid member")
		}
	})
}

// TestCallingConventions tests different calling conventions.
func TestCallingConventions(t *testing.T) {
	cif := &types.CallInterface{}

	t.Run("DefaultCall", func(t *testing.T) {
		err := PrepareCallInterface(cif, types.DefaultCall, types.VoidTypeDescriptor, nil)
		if err != nil {
			t.Errorf("DefaultCall failed: %v", err)
		}

		// Should resolve to platform-specific convention
		expected := types.DefaultConvention()
		if cif.Convention != expected {
			t.Errorf("Expected convention %d, got %d", expected, cif.Convention)
		}
	})

	t.Run("ExplicitUnix", func(t *testing.T) {
		err := PrepareCallInterface(cif, types.UnixCallingConvention, types.VoidTypeDescriptor, nil)
		if runtime.GOOS == "windows" {
			// On Windows, Unix convention might not work as expected
			t.Skip("Unix convention on Windows")
		}
		if err != nil {
			t.Errorf("UnixCallingConvention failed: %v", err)
		}
	})

	t.Run("ExplicitWindows", func(t *testing.T) {
		err := PrepareCallInterface(cif, types.WindowsCallingConvention, types.VoidTypeDescriptor, nil)
		if runtime.GOOS != "windows" {
			// On Unix, Windows convention might not work as expected
			t.Skip("Windows convention on Unix")
		}
		if err != nil {
			t.Errorf("WindowsCallingConvention failed: %v", err)
		}
	})

	t.Run("InvalidConvention", func(t *testing.T) {
		err := PrepareCallInterface(cif, types.CallingConvention(99), types.VoidTypeDescriptor, nil)
		if err == nil {
			t.Error("Expected error for invalid calling convention")
		}

		var ccErr *CallingConventionError
		if err != nil && err.(*CallingConventionError).Convention != 99 {
			t.Errorf("Expected CallingConventionError with value 99")
		}
		_ = ccErr
	})
}

// TestMultipleArguments tests functions with multiple arguments.
func TestMultipleArguments(t *testing.T) {
	cif := &types.CallInterface{}

	argTypes := []*types.TypeDescriptor{
		types.SInt32TypeDescriptor,
		types.SInt32TypeDescriptor,
		types.DoubleTypeDescriptor,
		types.PointerTypeDescriptor,
	}

	err := PrepareCallInterface(cif, types.DefaultCall, types.SInt32TypeDescriptor, argTypes)
	if err != nil {
		t.Fatalf("PrepareCallInterface failed: %v", err)
	}

	if cif.ArgCount != 4 {
		t.Errorf("Expected ArgCount=4, got %d", cif.ArgCount)
	}
}

// TestVoidFunction tests void return type.
func TestVoidFunction(t *testing.T) {
	cif := &types.CallInterface{}

	err := PrepareCallInterface(cif, types.DefaultCall, types.VoidTypeDescriptor, nil)
	if err != nil {
		t.Fatalf("PrepareCallInterface with void failed: %v", err)
	}

	if cif.ReturnType != types.VoidTypeDescriptor {
		t.Error("Expected void return type")
	}
}
