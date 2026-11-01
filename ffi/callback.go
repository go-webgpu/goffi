//go:build (linux || darwin || freebsd) && amd64

// Callback support (Unix amd64): Go function registration as C callbacks using
// pre-compiled assembly trampolines. Not a package comment — the package
// documentation lives in ffi.go.

package ffi

import (
	"math"
	"reflect"
	"strconv"
	"structs"
	"sync"
	"unsafe"

	archamd64 "github.com/go-webgpu/goffi/internal/arch/amd64"
)

// maxCallbacks is the maximum number of concurrent callbacks supported.
// This limit is determined by the number of trampoline entries in callback_amd64.s.
const maxCallbacks = 2000

// callbacks holds the global callback registry.
// The registry is thread-safe and stores all registered Go functions that can be
// called from C code. Functions are stored as reflect.Value to enable dynamic
// invocation with proper type checking and argument marshaling.
var callbacks struct {
	mu         sync.Mutex                  // Protects funcs, stackWords and count
	funcs      [maxCallbacks]reflect.Value // Registered callback functions
	stackWords [maxCallbacks]int           // Stack words of each callback's arguments
	count      int                         // Number of active callbacks
}

// callbackArgs represents the argument block passed from assembly to callbackWrap.
// This structure matches the memory layout created by the assembly trampoline code.
// The assembly code saves all CPU registers (both integer and SSE) into a contiguous
// memory block following this structure.
type callbackArgs struct {
	_      structs.HostLayout
	index  uintptr        // Callback index (0-1999)
	args   unsafe.Pointer // Pointer to register/stack argument block
	result uintptr        // Return value from Go callback, see callbackWrap
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
	stackWords := validateCallbackSignature(typ)

	// Register callback in global registry
	callbacks.mu.Lock()
	defer callbacks.mu.Unlock()

	if callbacks.count >= maxCallbacks {
		panic("ffi: callback limit reached (2000 callbacks maximum)")
	}

	idx := callbacks.count
	callbacks.funcs[idx] = val
	callbacks.stackWords[idx] = stackWords
	callbacks.count++

	// Return address to corresponding trampoline entry
	return trampolineEntryAddr(idx)
}

// validateCallbackSignature checks if a function type is valid for callbacks.
// This function enforces the constraints required by the FFI calling convention.
// It returns how many stack words the arguments occupy.
func validateCallbackSignature(typ reflect.Type) int {
	// Validate input arguments
	numIn := typ.NumIn()
	for i := 0; i < numIn; i++ {
		argType := typ.In(i)
		switch argType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Uintptr, reflect.Float32, reflect.Float64,
			reflect.Pointer, reflect.UnsafePointer, reflect.Bool, reflect.Struct:
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
			reflect.Pointer, reflect.UnsafePointer, reflect.Bool:
			// Valid return types
		default:
			panic("ffi: unsupported callback return type: " + retType.Kind().String())
		}
	default:
		panic("ffi: callbacks can only return zero or one value")
	}

	words := callbackStackWords(typ)
	if words > callbackMaxStackWords {
		panic("ffi: callback arguments need " + strconv.Itoa(words) + " stack words, at most " +
			strconv.Itoa(callbackMaxStackWords) + " are supported")
	}
	return words
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

// callbackWrap_call allows the calling of the ABIInternal wrapper
// which is required for runtime.cgocallback without the <ABIInternal>
// tag which is only allowed in the runtime.
// This closure is used inside callback_amd64.s to pass to crosscall2.
var callbackWrap_call = callbackWrap

// callbackWrap is called from assembly via crosscall2 to invoke the actual Go callback.
// This function handles:
//   - Looking up the callback function by index
//   - Marshaling C arguments to Go values using reflection
//   - Calling the Go function
//   - Marshaling the return value back to C format
//
// The assembly dispatcher (callback_amd64.s) routes through crosscall2 →
// runtime·load_g → runtime·cgocallback, which properly handles callbacks from
// both Go-managed threads and C-library-created threads (e.g. Metal's
// addCompletedHandler: dispatching on internal C threads).
//
// The assembly trampoline has already saved all CPU registers into a contiguous
// memory block pointed to by a.args. The memory layout follows the System V AMD64 ABI:
//   - Floats: XMM0-XMM7 (8 registers, 64 bytes); a float32 is the low 32 bits
//   - Integers: RDI, RSI, RDX, RCX, R8, R9 (6 registers, 48 bytes)
//   - Stack arguments follow in memory, one 8-byte slot per scalar
//
// The return value is stored in a.result in its native register representation
// (integers and pointers as is, float32 as math.Float32bits zero-extended,
// float64 as math.Float64bits). The dispatcher loads a.result into both RAX and
// XMM0; the register the caller does not read for the declared return type is
// caller-saved, so writing both is ABI-neutral.
//
// This function uses reflection to dynamically convert the register values into
// properly-typed Go values, which adds some overhead but provides type safety.
func callbackWrap(a *callbackArgs) {
	// Retrieve the registered callback function
	callbacks.mu.Lock()
	fn := callbacks.funcs[a.index]
	stackWords := callbacks.stackWords[a.index]
	callbacks.mu.Unlock()

	typ := fn.Type()
	numArgs := typ.NumIn()

	// Argument block layout (System V AMD64 ABI):
	// [XMM0-7: 64 bytes][RDI,RSI,RDX,RCX,R8,R9: 48 bytes][stack args...]
	// Arguments are assigned registers and stack words by the same allocator
	// the call path uses (internal/arch/amd64/layout_sysv.go), so an argument
	// that does not fit in the registers left is read from the stack whole.
	// The frame covers exactly the stack words the signature declares
	// (counted by validateCallbackSignature), which the C caller wrote.
	frame := unsafe.Slice((*uintptr)(a.args), callbackStackBase+stackWords)
	var alloc archamd64.SysVAllocator

	// Build argument slice for reflection Call
	args := make([]reflect.Value, numArgs)

	for i := 0; i < numArgs; i++ {
		argType := typ.In(i)
		p := placeCallbackArg(&alloc, argType)
		// word is the scalar argument's register or stack word. A float
		// occupies the low 32 bits (both in XMMn and in its 8-byte stack
		// slot), a double all 64.
		word := func() uintptr {
			if p.onStack {
				return frame[p.first]
			}
			return frame[p.idx[0]]
		}
		var val reflect.Value

		switch argType.Kind() {
		case reflect.Float32:
			// The caller passes a single-precision value, not a double:
			// reinterpret the low 32 bits, do not convert a float64.
			val = reflect.ValueOf(math.Float32frombits(uint32(word()))).Convert(argType)

		case reflect.Float64:
			val = reflect.ValueOf(math.Float64frombits(uint64(word()))).Convert(argType)

		case reflect.Bool:
			// Bool comes from integer register
			val = reflect.ValueOf(word() != 0)

		case reflect.Pointer:
			// Pointers come from integer registers.
			// The register contains the actual pointer value from C code.
			// Using unsafe.Pointer is necessary to convert uintptr (from register)
			// to Go pointer for reflection. This is safe because:
			// 1. The pointer came from C code which is responsible for its lifetime
			// 2. We're only using it for the duration of this callback invocation
			// 3. reflect.NewAt creates a proper typed pointer from the address
			w := word()
			// Double-indirection: reinterpret uintptr bits as pointer without
			// triggering checkptr arithmetic check (go.dev/issue/58625).
			ptr := *(*unsafe.Pointer)(unsafe.Pointer(&w))
			val = reflect.NewAt(argType.Elem(), ptr)

		case reflect.UnsafePointer:
			w := word()
			val = reflect.ValueOf(*(*unsafe.Pointer)(unsafe.Pointer(&w)))

		case reflect.Struct:
			// The struct's bytes as whole words: from its registers, one per
			// eightbyte, when they fit, otherwise from its stack words.
			sz := argType.Size()
			words := make([]uintptr, (sz+7)/8)
			if p.onStack {
				copy(words, frame[p.first:p.first+len(words)])
			} else {
				for k := range p.n {
					words[k] = frame[p.idx[k]]
				}
			}
			val = reflect.New(argType)
			if sz > 0 {
				copy(unsafe.Slice((*byte)(val.UnsafePointer()), sz),
					unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(words))), sz))
			}
			val = val.Elem()

		default:
			// All other integer types (int, uint, int32, etc.)
			value := word()
			val = reflect.NewAt(argType, unsafe.Pointer(&value)).Elem()
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
		case reflect.Pointer, reflect.UnsafePointer:
			a.result = ret.Pointer()
		case reflect.Float32:
			// Single precision in the low 32 bits, upper bits zero. The
			// dispatcher copies result into both RAX and XMM0, so a float
			// return reaches the caller in XMM0[31:0] (SysV psABI §3.2.3;
			// same as libffi's UNIX64_RET_XMM32, movd into %xmm0).
			a.result = uintptr(math.Float32bits(float32(ret.Float())))
		case reflect.Float64:
			a.result = uintptr(math.Float64bits(ret.Float()))
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

// Layout of the callback argument block (callback_amd64.s): XMM0-XMM7, then
// RDI, RSI, RDX, RCX, R8, R9, then the caller's stack arguments.
const (
	callbackSSERegs   = 8                   // XMM0-XMM7
	callbackStackBase = callbackSSERegs + 6 // first stack word
	// callbackMaxStackWords bounds the stack words a callback reads, as
	// MaxStackBytes bounds them for a call. They lie in the C caller's frame,
	// above the trampoline's saved registers; the C caller wrote every one
	// the signature declares.
	callbackMaxStackWords = archamd64.MaxStackBytes / 8
)

// callbackStructClasses classifies a struct argument like the call path does
// (internal/arch/amd64/layout_sysv.go): n eightbytes of the given classes, or
// memory for a struct over 16 bytes.
func callbackStructClasses(t reflect.Type) (sse [2]bool, n int, memory bool) {
	sz := t.Size()
	if sz > 16 {
		return sse, 0, true
	}
	n = int((sz + 7) / 8)
	for i := range n {
		start := uintptr(i) * 8
		sse[i] = classifyEightbyte(t, start, min(start+8, sz))
	}
	return sse, n, false
}

// countClasses returns how many of the first n eightbytes are INTEGER and SSE.
func countClasses(sse [2]bool, n int) (ngpr, nsse int) {
	for i := range n {
		if sse[i] {
			nsse++
		} else {
			ngpr++
		}
	}
	return ngpr, nsse
}

// classifyEightbyte reports whether the eightbyte [startOff, endOff) of a
// struct is SSE class: every scalar field that starts in it, at any nesting
// depth, is float32 or float64 (psABI §3.2.3: the classes of the fields in an
// eightbyte are merged and INTEGER wins). An eightbyte with no field is not SSE.
func classifyEightbyte(structType reflect.Type, startOff, endOff uintptr) bool {
	hasField, allFloat := false, true
	walkScalarFields(structType, 0, func(off uintptr, k reflect.Kind) bool {
		if off < startOff || off >= endOff {
			return true
		}
		hasField = true
		if k != reflect.Float32 && k != reflect.Float64 {
			allFloat = false
			return false
		}
		return true
	})
	return hasField && allFloat
}

// walkScalarFields calls fn with the offset (from base) and kind of every
// non-struct field of t, nested structs and arrays included, until fn returns
// false. It returns false when fn stopped the walk.
func walkScalarFields(t reflect.Type, base uintptr, fn func(off uintptr, k reflect.Kind) bool) bool {
	switch t.Kind() {
	case reflect.Struct:
		for i := range t.NumField() {
			f := t.Field(i)
			if !walkScalarFields(f.Type, base+f.Offset, fn) {
				return false
			}
		}
		return true
	case reflect.Array:
		elem := t.Elem()
		for i := range t.Len() {
			if !walkScalarFields(elem, base+uintptr(i)*elem.Size(), fn) {
				return false
			}
		}
		return true
	default:
		return fn(base, t.Kind())
	}
}

// callbackPlacement says where one callback argument is in the argument
// block: the frame index of each eightbyte in a register, or the frame index
// of its first stack word.
type callbackPlacement struct {
	n       int    // eightbytes in registers
	idx     [2]int // frame index of each eightbyte in a register
	onStack bool
	first   int // frame index of the first stack word when onStack
}

// placeCallbackArg assigns the next argument, of type t, its registers or
// stack words with the System V rules of the call path
// (internal/arch/amd64/layout_sysv.go).
func placeCallbackArg(alloc *archamd64.SysVAllocator, t reflect.Type) callbackPlacement {
	var (
		sse    [2]bool
		n      = 1
		memory bool
		size   uintptr = 8
		align  uintptr = 8
	)
	switch t.Kind() {
	case reflect.Float32, reflect.Float64:
		sse[0] = true
	case reflect.Struct:
		sse, n, memory = callbackStructClasses(t)
		size, align = t.Size(), uintptr(t.Align())
	}

	var p callbackPlacement
	if ngpr, nsse := countClasses(sse, n); !memory && alloc.Fits(ngpr, nsse) {
		p.n = n
		for k := range n {
			if sse[k] {
				p.idx[k] = alloc.SSE()
			} else {
				p.idx[k] = callbackSSERegs + alloc.GPR()
			}
		}
		return p
	}
	if size > 0 {
		p.onStack = true
		p.first = callbackStackBase + alloc.Stack(size, align)
	}
	return p
}

// callbackStackWords returns how many stack words the arguments of a callback
// of type typ occupy.
func callbackStackWords(typ reflect.Type) int {
	var alloc archamd64.SysVAllocator
	for i := range typ.NumIn() {
		placeCallbackArg(&alloc, typ.In(i))
	}
	return int(alloc.StackBytes() / 8)
}
