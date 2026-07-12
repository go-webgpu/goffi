// Package ffi provides a Foreign Function Interface for calling C functions from Go
// without CGO. It enables direct calls to C libraries with full type safety and
// platform abstraction.
//
// # Overview
//
// This package allows you to:
//   - Load dynamic libraries (LoadLibrary)
//   - Get function pointers (GetSymbol)
//   - Prepare function call interfaces (PrepareCallInterface)
//   - Execute C function calls (CallFunction)
//
// # Basic Usage
//
//	// Load a library
//	handle, err := ffi.LoadLibrary("libm.so.6")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get a function pointer
//	sqrtPtr, err := ffi.GetSymbol(handle, "sqrt")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Prepare call interface
//	var cif types.CallInterface
//	err = ffi.PrepareCallInterface(
//	    &cif,
//	    types.DefaultCall,
//	    types.DoubleTypeDescriptor,
//	    []*types.TypeDescriptor{types.DoubleTypeDescriptor},
//	)
//
//	// Call the function
//	var result float64
//	arg := 16.0
//	err = ffi.CallFunction(
//	    &cif,
//	    sqrtPtr,
//	    unsafe.Pointer(&result),
//	    []unsafe.Pointer{unsafe.Pointer(&arg)},
//	)
//	// result is now 4.0
//
// # Supported Platforms
//
//   - Linux AMD64 (System V ABI)
//   - Windows AMD64 (Win64 ABI)
//   - macOS AMD64 (planned)
//   - ARM64 (planned)
//
// # Performance
//
// This implementation uses hand-optimized assembly for each platform's calling
// convention. Overhead is approximately 50-60ns per call, which is negligible
// for most use cases (e.g., WebGPU rendering).
//
// # Safety
//
// While this package uses unsafe.Pointer internally, the public API validates
// all inputs and provides type-safe wrappers. Users should ensure:
//   - Argument types match the C function signature exactly
//   - Pointers remain valid during the call (use runtime.KeepAlive if needed)
//   - Return value buffer is large enough for the result
//
// # Thread Safety
//
// This package follows Go's standard library conventions for concurrent access:
//   - PrepareCallInterface and CallFunction are safe to call concurrently with different CallInterface instances
//   - DO NOT use the same CallInterface from multiple goroutines simultaneously without external synchronization
//   - Library handles (from LoadLibrary) are safe to use concurrently for read operations (GetSymbol)
//   - DO NOT call FreeLibrary while other goroutines are using GetSymbol on the same handle
//   - Similar to io.Reader: methods are not inherently thread-safe; synchronization is caller's responsibility
//
// Race detector is not supported for zero-CGO libraries (race detector requires CGO_ENABLED=1,
// which conflicts with our fakecgo implementation using build tag !cgo). This is a fundamental
// limitation of the zero-CGO approach. However, this library contains no data races in its
// internal implementation - all shared state (Registry, TypeDescriptors) is initialized once
// at startup and accessed read-only thereafter.
//
// # Zero Dependencies
//
// This package has zero external dependencies (except for internal/fakecgo on Linux).
// All FFI logic is implemented in pure Go and assembly.
package ffi

import (
	"context"
	"errors"
	"syscall"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// PrepareCallInterface prepares a function call interface for calling a C function.
//
// This function initializes the CallInterface structure with the necessary metadata
// for making FFI calls. It must be called before CallFunction.
//
// Parameters:
//   - cif: Pointer to CallInterface structure to initialize (must not be nil)
//   - convention: Calling convention (types.DefaultCall, types.CDecl, types.StdCall, etc.)
//   - returnType: Type descriptor for return value (use types.VoidTypeDescriptor for void)
//   - argTypes: Slice of type descriptors for each argument (nil or empty slice for no arguments)
//
// Returns:
//   - nil on success
//   - ErrInvalidCallInterface if parameters are invalid
//   - Other errors if type validation or platform preparation fails
//
// Example:
//
//	var cif types.CallInterface
//	err := ffi.PrepareCallInterface(
//	    &cif,
//	    types.DefaultCall,
//	    types.Int32TypeDescriptor,
//	    []*types.TypeDescriptor{
//	        types.PointerTypeDescriptor,
//	        types.Int32TypeDescriptor,
//	    },
//	)
//
// For functions with no arguments, pass nil or an empty slice for argTypes:
//
//	err := ffi.PrepareCallInterface(&cif, types.DefaultCall, types.VoidTypeDescriptor, nil)
func PrepareCallInterface(
	cif *types.CallInterface,
	convention types.CallingConvention,
	returnType *types.TypeDescriptor,
	argTypes []*types.TypeDescriptor,
) error {
	if cif == nil {
		return &InvalidCallInterfaceError{
			Field:  "cif",
			Reason: "must not be nil",
			Index:  -1,
		}
	}
	if returnType == nil {
		return &InvalidCallInterfaceError{
			Field:  "returnType",
			Reason: "must not be nil",
			Index:  -1,
		}
	}

	argCount := len(argTypes)
	return prepareCallInterfaceCore(cif, convention, argCount, returnType, argTypes)
}

// PrepareVariadicCallInterface prepares a call interface for a C variadic function.
//
// nfixedargs is the count of fixed parameters before '...' in the C prototype.
// argTypes must contain ALL arguments (fixed + variadic) for this specific call.
// A new CIF must be prepared for each unique combination of variadic argument types.
//
// On Apple ARM64, variadic arguments are forced to the stack per Apple's AAPCS64
// extension. The register allocators are exhausted at the fixed/variadic boundary
// so that variadic arguments land on the stack even when registers are available.
// On all other platforms, this function behaves identically to PrepareCallInterface.
//
// Example:
//
//	// Prepare for: int64_t sum_variadic(int64_t count, ...)
//	// Called with count=3 and three int64_t variadic args.
//	var cif types.CallInterface
//	err := ffi.PrepareVariadicCallInterface(
//	    &cif,
//	    types.DefaultCall,
//	    1, // nfixedargs: only 'count' is fixed
//	    types.SInt64TypeDescriptor,
//	    []*types.TypeDescriptor{
//	        types.SInt64TypeDescriptor, // count
//	        types.SInt64TypeDescriptor, // arg1 (variadic)
//	        types.SInt64TypeDescriptor, // arg2 (variadic)
//	        types.SInt64TypeDescriptor, // arg3 (variadic)
//	    },
//	)
func PrepareVariadicCallInterface(
	cif *types.CallInterface,
	convention types.CallingConvention,
	nfixedargs int,
	returnType *types.TypeDescriptor,
	argTypes []*types.TypeDescriptor,
) error {
	if nfixedargs < 0 {
		return errors.New("goffi: nfixedargs must be non-negative")
	}
	if nfixedargs > len(argTypes) {
		return errors.New("goffi: nfixedargs exceeds total argument count")
	}
	if err := PrepareCallInterface(cif, convention, returnType, argTypes); err != nil {
		return err
	}
	cif.FixedArgCount = nfixedargs
	return nil
}

// CallFunctionContext executes a C function call with context support.
//
// This function performs the actual FFI call to the C function, handling all
// platform-specific calling convention details automatically. It checks the
// context before executing to prevent starting expensive operations when the
// context is already cancelled or has exceeded its deadline.
//
// errno is captured inside the assembly trampoline immediately after the C
// function returns, before the Go runtime can migrate the goroutine to a
// different OS thread. On Windows, errno is always 0 (use syscall.GetLastError()
// for Win32 error codes).
//
// Parameters:
//   - ctx: Context for cancellation and timeout control (use context.Background() if not needed)
//   - cif: Prepared call interface (from PrepareCallInterface)
//   - fn: Function pointer obtained from GetSymbol (must not be nil)
//   - rvalue: Pointer to buffer for return value (can be nil for void functions)
//   - avalue: Slice of pointers to argument values (length must match argCount from PrepareCallInterface)
//
// Returns:
//   - errno: C errno captured after the call (0 on success or Windows)
//   - err: nil on success; ctx.Err() if context cancelled; ErrInvalidCallInterface if cif or fn is nil
//
// Example:
//
//	// Call with timeout
//	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
//	defer cancel()
//
//	var result float64
//	arg := 16.0
//	errno, err := ffi.CallFunctionContext(
//	    ctx,
//	    &cif,
//	    sqrtPtr,
//	    unsafe.Pointer(&result),
//	    []unsafe.Pointer{unsafe.Pointer(&arg)},
//	)
//	if err == context.DeadlineExceeded {
//	    log.Println("Call timed out")
//	}
//
// Note:
//   - Context cancellation check occurs BEFORE the call to prevent starting
//     expensive operations when the context is already cancelled.
//   - Once the C function starts executing, it CANNOT be interrupted mid-flight.
//   - For cancellable operations, the C library itself must support cancellation.
//
// Safety:
//   - All argument pointers must remain valid during the call
//   - Return value buffer must be large enough for the result type
//   - Use runtime.KeepAlive() if needed to prevent premature GC of arguments
//   - Use runtime.Pinner to pin pointers under a moving GC
func CallFunctionContext(
	ctx context.Context,
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
) (syscall.Errno, error) {
	// Check context before expensive call
	if ctxErr := ctx.Err(); ctxErr != nil {
		return 0, ctxErr
	}

	if cif == nil {
		return 0, &InvalidCallInterfaceError{
			Field:  "cif",
			Reason: "must not be nil",
			Index:  -1,
		}
	}
	if fn == nil {
		return 0, &InvalidCallInterfaceError{
			Field:  "fn",
			Reason: "function pointer must not be nil",
			Index:  -1,
		}
	}

	cerrno, err := executeFunction(cif, fn, rvalue, avalue)
	return syscall.Errno(cerrno), err
}

// CallFunction executes a C function call without context support.
//
// This is equivalent to CallFunctionContext(context.Background(), cif, fn, rvalue, avalue).
// For operations that need cancellation or timeout control, use CallFunctionContext instead.
//
// errno is captured inside the assembly trampoline immediately after the C
// function returns, before the Go runtime can migrate the goroutine to a
// different OS thread. On Windows, errno is always 0.
//
// Parameters:
//   - cif: Prepared call interface (from PrepareCallInterface)
//   - fn: Function pointer obtained from GetSymbol (must not be nil)
//   - rvalue: Pointer to buffer for return value (can be nil for void functions)
//   - avalue: Slice of pointers to argument values (length must match argCount from PrepareCallInterface)
//
// Returns:
//   - errno: C errno captured after the call (0 on success or Windows)
//   - err: nil on success; ErrInvalidCallInterface if cif or fn is nil
//
// Example:
//
//	// Calling open(2) and checking errno on failure:
//	errno, err := ffi.CallFunction(
//	    &cif,
//	    openFn,
//	    unsafe.Pointer(&result),
//	    []unsafe.Pointer{unsafe.Pointer(&pathPtr), unsafe.Pointer(&flags)},
//	)
//	if result == -1 {
//	    log.Printf("open failed: %v", errno)
//	}
//
//	// When errno is not needed:
//	_, err := ffi.CallFunction(&cif, strlenFn, unsafe.Pointer(&result), avalue)
//
// For context-aware calls with timeout support, see CallFunctionContext.
func CallFunction(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
) (syscall.Errno, error) {
	return CallFunctionContext(context.Background(), cif, fn, rvalue, avalue)
}
