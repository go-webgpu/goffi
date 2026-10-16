//go:build ((linux && !android) || darwin || freebsd) && arm64

// Package ffi provides callback support for Foreign Function Interface (ARM64 Unix version).
// This file implements Go function registration as C callbacks using
// pre-compiled assembly trampolines for optimal performance.
package ffi

import (
	"math"
	"reflect"
	"runtime"
	"structs"
	"sync"
	"unsafe"

	archarm64 "github.com/go-webgpu/goffi/internal/arch/arm64"
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
	_      structs.HostLayout
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

	return trampolineEntryAddr(idx)
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
			reflect.Pointer, reflect.UnsafePointer, reflect.Bool:
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
			reflect.Pointer, reflect.UnsafePointer, reflect.Bool:
			// Valid return types
		default:
			panic("ffi: unsupported callback return type: " + retType.Kind().String())
		}
	default:
		panic("ffi: callbacks can only return zero or one value")
	}
}

// trampolineEntryAddr calculates the address of a specific trampoline entry.
// Each trampoline entry is 8 bytes on ARM64 (MOVD $index, R12 = 4 bytes + B dispatcher = 4 bytes).
func trampolineEntryAddr(i int) uintptr {
	const entrySize = 8 // ARM64: MOVD (4 bytes) + B (4 bytes)
	return trampolineBaseAddr + uintptr(i*entrySize)
}

// callbackWrap_call allows the calling of the ABIInternal wrapper
// which is required for runtime.cgocallback without the <ABIInternal>
// tag which is only allowed in the runtime.
// This closure is used inside callback_arm64.s to pass to crosscall2.
var callbackWrap_call = callbackWrap

// callbackWrap is called from assembly via crosscall2 to invoke the actual Go callback.
//
// The assembly dispatcher (callback_arm64.s) routes through crosscall2 →
// runtime·load_g → runtime·cgocallback, which properly handles callbacks from
// both Go-managed threads and C-library-created threads (e.g. Metal's
// addCompletedHandler: dispatching on internal C threads).
//
// ARM64 AAPCS64 argument block layout:
//   - Floats: D0-D7 (8 registers, 64 bytes); a float32 is S<n>, the low 32 bits
//   - Integers: X0-X7 (8 registers, 64 bytes)
//   - Stack arguments follow in memory: one 8-byte slot per scalar on Linux and
//     FreeBSD, natural size and alignment on Apple platforms (see
//     internal/arch/arm64/layout.go).
//
// The return value is stored in a.result in its native register representation
// (integers and pointers as is, float32 as math.Float32bits zero-extended,
// float64 as math.Float64bits). The dispatcher loads a.result into both X0 and
// D0.
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

	// Stack arguments start right after the saved registers; their offsets
	// follow the same rules as the call path (8-byte slots on Linux and
	// FreeBSD, natural size and alignment on Apple platforms).
	stackBase := unsafe.Pointer(&frame[numFloatRegs+numIntRegs])
	stack := archarm64.NewStackAllocator(runtime.GOOS)

	var floatIdx, intIdx int

	// nextSlot returns the address of the next argument of class float or
	// integer: its saved register while one is left, otherwise its stack
	// location. Values narrower than a register sit in its low bytes.
	nextSlot := func(isFloat bool, t reflect.Type) unsafe.Pointer {
		if isFloat && floatIdx < numFloatRegs {
			p := unsafe.Pointer(&frame[floatIdx])
			floatIdx++
			return p
		}
		if !isFloat && intIdx < numIntRegs {
			p := unsafe.Pointer(&frame[numFloatRegs+intIdx])
			intIdx++
			return p
		}
		return unsafe.Add(stackBase, stack.Alloc(uintptr(t.Align()), t.Size()))
	}

	args := make([]reflect.Value, numArgs)

	for i := 0; i < numArgs; i++ {
		argType := typ.In(i)
		var val reflect.Value

		switch argType.Kind() {
		case reflect.Float32:
			// A float arrives in S<n>, the low 32 bits of the saved D<n>.
			// Reinterpret those bits, do not convert a double.
			p := nextSlot(true, argType)
			val = reflect.ValueOf(math.Float32frombits(*(*uint32)(p))).Convert(argType)

		case reflect.Float64:
			p := nextSlot(true, argType)
			val = reflect.ValueOf(math.Float64frombits(*(*uint64)(p))).Convert(argType)

		case reflect.Bool:
			// A C _Bool is one byte; the rest of its register is unspecified.
			p := nextSlot(false, argType)
			val = reflect.ValueOf(*(*uint8)(p) != 0)

		case reflect.Pointer:
			p := nextSlot(false, argType)
			val = reflect.NewAt(argType.Elem(), *(*unsafe.Pointer)(p))

		case reflect.UnsafePointer:
			p := nextSlot(false, argType)
			val = reflect.ValueOf(*(*unsafe.Pointer)(p))

		default:
			p := nextSlot(false, argType)
			val = reflect.NewAt(argType, p).Elem()
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
		case reflect.Pointer, reflect.UnsafePointer:
			a.result = ret.Pointer()
		case reflect.Float32:
			// Single precision in the low 32 bits, upper bits zero. The
			// dispatcher copies result into both X0 and D0, so a float return
			// reaches the caller in S0 (as libffi's closure does: ldr s0).
			a.result = uintptr(math.Float32bits(float32(ret.Float())))
		case reflect.Float64:
			a.result = uintptr(math.Float64bits(ret.Float()))
		}
	}
}

// trampolineBaseAddr is the address of the callback assembly trampoline table.
//
//go:linkname _callbackTrampoline github.com/go-webgpu/goffi/ffi.callbackTrampoline
var _callbackTrampoline byte
var trampolineBaseAddr = uintptr(unsafe.Pointer(&_callbackTrampoline))
