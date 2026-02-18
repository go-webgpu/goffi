//go:build (linux || darwin) && amd64

// Package ffi provides callback support for Foreign Function Interface (Unix version).
// This file implements Go function registration as C callbacks using
// pre-compiled assembly trampolines for optimal performance.
package ffi

import (
	"reflect"
	"sync"
	"unsafe"
)

// maxCallbacks is the maximum number of concurrent callbacks supported.
// This limit is determined by the number of trampoline entries in callback_amd64.s.
const maxCallbacks = 2000

// callbacks holds the global callback registry.
// The registry is thread-safe and stores all registered Go functions that can be
// called from C code. Functions are stored as reflect.Value to enable dynamic
// invocation with proper type checking and argument marshaling.
var callbacks struct {
	mu    sync.Mutex                  // Protects funcs and count
	funcs [maxCallbacks]reflect.Value // Registered callback functions
	count int                         // Number of active callbacks
}

// callbackArgs represents the argument block passed from assembly to callbackWrap.
// This structure matches the memory layout created by the assembly trampoline code.
// The assembly code saves all CPU registers (both integer and SSE) into a contiguous
// memory block following this structure.
type callbackArgs struct {
	index  uintptr        // Callback index (0-1999)
	args   unsafe.Pointer // Pointer to register/stack argument block
	result uintptr        // Return value from Go callback
}

// NewCallback registers a Go function as a C callback and returns a function pointer.
// The returned uintptr can be passed to C code as a callback function pointer.
//
// Requirements:
//   - fn must be a function (not nil)
//   - fn can have multiple arguments of basic types (int, float, pointer, etc.)
//   - fn can return at most one value of basic type
//   - Complex types (string, slice, map, chan, interface) are not supported
//   - Maximum 2000 callbacks can be registered (program lifetime limit)
//
// Memory Management:
//   - Callbacks are never freed (stored in global registry)
//   - This prevents GC from collecting callback data while C code uses it
//   - For applications with dynamic callback creation, consider callback pools
//
// Usage Example:
//
//	func myCallback(x int, y float64) int {
//	    return x + int(y)
//	}
//
//	callbackPtr := ffi.NewCallback(myCallback)
//	// Pass callbackPtr to C code as function pointer
//
// Using unsafe.Pointer is necessary here as we're creating a function pointer
// that C code can call. The pointer is obtained from the assembly trampoline table
// and is guaranteed to be valid for the program lifetime.
func NewCallback(fn any) uintptr {
	if fn == nil {
		panic("ffi: callback function must not be nil")
	}

	val := reflect.ValueOf(fn)
	if val.Kind() != reflect.Func {
		panic("ffi: callback must be a function")
	}

	// Validate function signature
	typ := val.Type()
	validateCallbackSignature(typ)

	// Register callback in global registry
	callbacks.mu.Lock()
	defer callbacks.mu.Unlock()

	if callbacks.count >= maxCallbacks {
		panic("ffi: callback limit reached (2000 callbacks maximum)")
	}

	idx := callbacks.count
	callbacks.funcs[idx] = val
	callbacks.count++

	// Return address to corresponding trampoline entry
	return trampolineEntryAddr(idx)
}

// validateCallbackSignature checks if a function type is valid for callbacks.
// This function enforces the constraints required by the FFI calling convention.
func validateCallbackSignature(typ reflect.Type) {
	// Validate input arguments
	numIn := typ.NumIn()
	for i := 0; i < numIn; i++ {
		argType := typ.In(i)
		switch argType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Uintptr, reflect.Float32, reflect.Float64,
			reflect.Ptr, reflect.UnsafePointer, reflect.Bool:
			// Valid types
		default:
			panic("ffi: unsupported callback argument type: " + argType.Kind().String())
		}
	}

	// Validate return value
	numOut := typ.NumOut()
	switch numOut {
	case 0:
		// Void return is valid
	case 1:
		retType := typ.Out(0)
		switch retType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Uintptr, reflect.Float32, reflect.Float64,
			reflect.Ptr, reflect.UnsafePointer, reflect.Bool:
			// Valid return types
		default:
			panic("ffi: unsupported callback return type: " + retType.Kind().String())
		}
	default:
		panic("ffi: callbacks can only return zero or one value")
	}
}

// trampolineEntryAddr calculates the address of a specific trampoline entry.
// Each trampoline entry is 5 bytes (CALL instruction) on AMD64.
// The calculation is: base_address + (index * entry_size).
//
// This function is called by NewCallback to get the C-callable function pointer
// for a registered Go callback.
func trampolineEntryAddr(i int) uintptr {
	const entrySize = 5 // AMD64: CALL instruction = 5 bytes
	return trampolineBaseAddr + uintptr(i*entrySize)
}

// callbackWrap is called from assembly trampolines to invoke the actual Go callback.
// This function handles:
//   - Looking up the callback function by index
//   - Marshaling C arguments to Go values using reflection
//   - Calling the Go function
//   - Marshaling the return value back to C format
//
// The assembly trampoline (callback_amd64.s) has already saved all CPU registers
// into a contiguous memory block pointed to by a.args. The memory layout follows
// the System V AMD64 ABI:
//   - Floats: XMM0-XMM7 (8 registers, 64 bytes)
//   - Integers: RDI, RSI, RDX, RCX, R8, R9 (6 registers, 48 bytes)
//   - Stack arguments follow in memory
//
// This function uses reflection to dynamically convert the register values into
// properly-typed Go values, which adds some overhead but provides type safety.
func callbackWrap(a *callbackArgs) {
	// Retrieve the registered callback function
	callbacks.mu.Lock()
	fn := callbacks.funcs[a.index]
	callbacks.mu.Unlock()

	typ := fn.Type()
	numArgs := typ.NumIn()

	// Argument block layout (System V AMD64 ABI):
	// [XMM0-7: 64 bytes][RDI,RSI,RDX,RCX,R8,R9: 48 bytes][stack args...]
	const (
		numFloatRegs = 8 // XMM0-XMM7
		numIntRegs   = 6 // RDI, RSI, RDX, RCX, R8, R9
	)

	// Cast args pointer to array for easy indexing
	// Each register is 8 bytes (64-bit)
	frame := (*[128]uintptr)(a.args) // Large enough for registers + reasonable stack args

	var floatIdx int                      // Current float register index (0-7)
	var intIdx int                        // Current integer register index (0-5)
	stackIdx := numFloatRegs + numIntRegs // Stack arguments start after registers

	// Build argument slice for reflection Call
	args := make([]reflect.Value, numArgs)

	for i := 0; i < numArgs; i++ {
		argType := typ.In(i)
		var val reflect.Value

		switch argType.Kind() {
		case reflect.Float32:
			// Float32 comes from XMM register (stored as float64)
			if floatIdx < numFloatRegs {
				// Read as uintptr, reinterpret as float64 bits
				bits := frame[floatIdx]
				f64 := *(*float64)(unsafe.Pointer(&bits))
				val = reflect.ValueOf(float32(f64))
				floatIdx++
			} else {
				// From stack
				bits := frame[stackIdx]
				f64 := *(*float64)(unsafe.Pointer(&bits))
				val = reflect.ValueOf(float32(f64))
				stackIdx++
			}

		case reflect.Float64:
			// Float64 comes from XMM register
			if floatIdx < numFloatRegs {
				bits := frame[floatIdx]
				f64 := *(*float64)(unsafe.Pointer(&bits))
				val = reflect.ValueOf(f64)
				floatIdx++
			} else {
				bits := frame[stackIdx]
				f64 := *(*float64)(unsafe.Pointer(&bits))
				val = reflect.ValueOf(f64)
				stackIdx++
			}

		case reflect.Bool:
			// Bool comes from integer register
			if intIdx < numIntRegs {
				pos := numFloatRegs + intIdx
				val = reflect.ValueOf(frame[pos] != 0)
				intIdx++
			} else {
				val = reflect.ValueOf(frame[stackIdx] != 0)
				stackIdx++
			}

		case reflect.Ptr:
			// Pointers come from integer registers.
			// The register contains the actual pointer value from C code.
			// Using unsafe.Pointer is necessary to convert uintptr (from register)
			// to Go pointer for reflection. This is safe because:
			// 1. The pointer came from C code which is responsible for its lifetime
			// 2. We're only using it for the duration of this callback invocation
			// 3. reflect.NewAt creates a proper typed pointer from the address
			if intIdx < numIntRegs {
				pos := numFloatRegs + intIdx
				//nolint:govet,gosec // G103: Converting uintptr from CPU register to pointer for FFI callback argument
				ptr := unsafe.Pointer(frame[pos])
				val = reflect.NewAt(argType.Elem(), ptr)
				intIdx++
			} else {
				//nolint:govet,gosec // G103: Converting uintptr from stack to pointer for FFI callback argument
				ptr := unsafe.Pointer(frame[stackIdx])
				val = reflect.NewAt(argType.Elem(), ptr)
				stackIdx++
			}

		case reflect.UnsafePointer:
			// UnsafePointer comes from integer registers.
			// Converting uintptr to unsafe.Pointer is necessary for FFI interop.
			if intIdx < numIntRegs {
				pos := numFloatRegs + intIdx
				//nolint:govet,gosec // G103: Converting uintptr from register to unsafe.Pointer for FFI
				val = reflect.ValueOf(unsafe.Pointer(frame[pos]))
				intIdx++
			} else {
				//nolint:govet,gosec // G103: Converting uintptr from stack to unsafe.Pointer for FFI
				val = reflect.ValueOf(unsafe.Pointer(frame[stackIdx]))
				stackIdx++
			}

		default:
			// All other integer types (int, uint, int32, etc.)
			if intIdx < numIntRegs {
				pos := numFloatRegs + intIdx
				val = reflect.NewAt(argType, unsafe.Pointer(&frame[pos])).Elem()
				intIdx++
			} else {
				val = reflect.NewAt(argType, unsafe.Pointer(&frame[stackIdx])).Elem()
				stackIdx++
			}
		}

		args[i] = val
	}

	// Call the Go function
	results := fn.Call(args)

	// Marshal return value if present
	if len(results) > 0 {
		ret := results[0]
		switch ret.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			a.result = uintptr(ret.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			a.result = uintptr(ret.Uint())
		case reflect.Bool:
			if ret.Bool() {
				a.result = 1
			} else {
				a.result = 0
			}
		case reflect.Ptr, reflect.UnsafePointer:
			a.result = ret.Pointer()
		case reflect.Float32, reflect.Float64:
			// For float returns, store the bits as uintptr
			// The assembly code will move this to XMM0 for return
			f64 := ret.Float()
			a.result = *(*uintptr)(unsafe.Pointer(&f64))
		}
	}
}

// trampolineBaseAddr is the address of the callback assembly trampoline table.
// This variable is linked to the callbackTrampoline symbol defined in callback_amd64.s.
// Using //go:linkname allows us to access the assembly symbol from Go code.
//
//go:linkname _callbackTrampoline github.com/go-webgpu/goffi/ffi.callbackTrampoline
var _callbackTrampoline byte
var trampolineBaseAddr = uintptr(unsafe.Pointer(&_callbackTrampoline))
