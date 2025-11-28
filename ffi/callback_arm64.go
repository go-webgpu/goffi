//go:build (linux || darwin) && arm64

// Package ffi provides callback support for Foreign Function Interface (ARM64 Unix version).
// This file implements Go function registration as C callbacks using
// pre-compiled assembly trampolines for optimal performance.
package ffi

import (
	"reflect"
	"sync"
	"unsafe"
)

// maxCallbacks is the maximum number of concurrent callbacks supported.
// This limit is determined by the number of trampoline entries in callback_arm64.s.
const maxCallbacks = 2000

// callbacks holds the global callback registry.
var callbacks struct {
	mu    sync.Mutex
	funcs [maxCallbacks]reflect.Value
	count int
}

// callbackArgs represents the argument block passed from assembly to callbackWrap.
// ARM64 AAPCS64 layout: D0-D7 (float), X0-X7 (integer)
type callbackArgs struct {
	index  uintptr        // Callback index (0-1999)
	args   unsafe.Pointer // Pointer to register/stack argument block
	result uintptr        // Return value from Go callback
}

// NewCallback registers a Go function as a C callback and returns a function pointer.
func NewCallback(fn any) uintptr {
	if fn == nil {
		panic("ffi: callback function must not be nil")
	}

	val := reflect.ValueOf(fn)
	if val.Kind() != reflect.Func {
		panic("ffi: callback must be a function")
	}

	typ := val.Type()
	validateCallbackSignature(typ)

	callbacks.mu.Lock()
	defer callbacks.mu.Unlock()

	if callbacks.count >= maxCallbacks {
		panic("ffi: callback limit reached (2000 callbacks maximum)")
	}

	idx := callbacks.count
	callbacks.funcs[idx] = val
	callbacks.count++

	return callbackasmAddr(idx)
}

// validateCallbackSignature checks if a function type is valid for callbacks.
func validateCallbackSignature(typ reflect.Type) {
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

// callbackasmAddr calculates the address of a specific trampoline entry.
// Each trampoline entry is 8 bytes on ARM64.
func callbackasmAddr(i int) uintptr {
	const entrySize = 8 // ARM64: BL + MOV instructions
	return callbackasmABI0 + uintptr(i*entrySize)
}

// callbackWrap is called from assembly trampolines to invoke the actual Go callback.
// ARM64 AAPCS64 argument block layout:
//   - Floats: D0-D7 (8 registers, 64 bytes)
//   - Integers: X0-X7 (8 registers, 64 bytes)
//   - Stack arguments follow in memory
func callbackWrap(a *callbackArgs) {
	callbacks.mu.Lock()
	fn := callbacks.funcs[a.index]
	callbacks.mu.Unlock()

	typ := fn.Type()
	numArgs := typ.NumIn()

	const (
		numFloatRegs = 8 // D0-D7
		numIntRegs   = 8 // X0-X7
	)

	frame := (*[128]uintptr)(a.args)

	var floatIdx int
	var intIdx int
	stackIdx := numFloatRegs + numIntRegs

	args := make([]reflect.Value, numArgs)

	for i := 0; i < numArgs; i++ {
		argType := typ.In(i)
		var val reflect.Value

		switch argType.Kind() {
		case reflect.Float32:
			if floatIdx < numFloatRegs {
				bits := frame[floatIdx]
				f64 := *(*float64)(unsafe.Pointer(&bits))
				val = reflect.ValueOf(float32(f64))
				floatIdx++
			} else {
				bits := frame[stackIdx]
				f64 := *(*float64)(unsafe.Pointer(&bits))
				val = reflect.ValueOf(float32(f64))
				stackIdx++
			}

		case reflect.Float64:
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
			if intIdx < numIntRegs {
				pos := numFloatRegs + intIdx
				val = reflect.ValueOf(frame[pos] != 0)
				intIdx++
			} else {
				val = reflect.ValueOf(frame[stackIdx] != 0)
				stackIdx++
			}

		case reflect.Ptr:
			if intIdx < numIntRegs {
				pos := numFloatRegs + intIdx
				//nolint:govet,gosec // G103: FFI callback argument
				ptr := unsafe.Pointer(frame[pos])
				val = reflect.NewAt(argType.Elem(), ptr)
				intIdx++
			} else {
				//nolint:govet,gosec // G103: FFI callback argument
				ptr := unsafe.Pointer(frame[stackIdx])
				val = reflect.NewAt(argType.Elem(), ptr)
				stackIdx++
			}

		case reflect.UnsafePointer:
			if intIdx < numIntRegs {
				pos := numFloatRegs + intIdx
				//nolint:govet,gosec // G103: FFI callback argument
				val = reflect.ValueOf(unsafe.Pointer(frame[pos]))
				intIdx++
			} else {
				//nolint:govet,gosec // G103: FFI callback argument
				val = reflect.ValueOf(unsafe.Pointer(frame[stackIdx]))
				stackIdx++
			}

		default:
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

	results := fn.Call(args)

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
			f64 := ret.Float()
			a.result = *(*uintptr)(unsafe.Pointer(&f64))
		}
	}
}

// callbackasmABI0 is the address of the callback assembly trampoline table.
//
//go:linkname __callbackasm callbackasm
var __callbackasm byte
var callbackasmABI0 = uintptr(unsafe.Pointer(&__callbackasm))
