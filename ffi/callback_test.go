//go:build (linux || darwin) && (amd64 || arm64)

package ffi

import (
	"runtime"
	"sync"
	"testing"
	"unsafe"
)

const callbackFloatRegCount = 8

func callbackEntrySize() uintptr {
	return callbackasmAddr(1) - callbackasmAddr(0)
}

func callbackIndex(ptr uintptr) uintptr {
	entrySize := callbackEntrySize()
	if entrySize == 0 {
		return 0
	}
	return (ptr - callbackasmABI0) / entrySize
}

func callbackIntRegCount() int {
	if runtime.GOARCH == "arm64" {
		return 8
	}
	return 6
}

func callbackIntRegIndex(i int) int {
	return callbackFloatRegCount + i
}

func callbackStackIndex(i int) int {
	return callbackFloatRegCount + callbackIntRegCount() + i
}

// Test basic callback registration.
func TestNewCallback_BasicRegistration(t *testing.T) {
	callback := func() {
		// Simple callback for registration test
	}

	ptr := NewCallback(callback)

	if ptr == 0 {
		t.Fatal("NewCallback returned nil pointer")
	}

	// Verify pointer is within expected range
	entrySize := callbackEntrySize()
	baseAddr := callbackasmABI0
	maxAddr := callbackasmABI0 + uintptr(maxCallbacks)*entrySize

	if ptr < baseAddr || ptr >= maxAddr {
		t.Errorf("Callback pointer %x outside expected range [%x, %x)", ptr, baseAddr, maxAddr)
	}
}

// Test callback with integer arguments.
func TestCallback_IntegerArgs(t *testing.T) {
	var result int
	callback := func(a int, b int) {
		result = a + b
	}

	ptr := NewCallback(callback)
	if ptr == 0 {
		t.Fatal("NewCallback returned nil pointer")
	}

	// Simulate C calling the callback
	// We'll invoke callbackWrap directly to test argument marshaling
	idx := callbackIndex(ptr)

	// Create argument frame (System V AMD64 ABI)
	// Layout: [XMM0-7][RDI,RSI,RDX,RCX,R8,R9][stack...]
	var frame [128]uintptr
	frame[callbackIntRegIndex(0)] = 10 // first int arg
	frame[callbackIntRegIndex(1)] = 20 // second int arg

	args := &callbackArgs{
		index: idx,
		args:  unsafe.Pointer(&frame),
	}

	callbackWrap(args)

	if result != 30 {
		t.Errorf("Expected result 30, got %d", result)
	}
}

// Test callback with integer return value.
func TestCallback_IntegerReturn(t *testing.T) {
	callback := func(a int, b int) int {
		return a * b
	}

	ptr := NewCallback(callback)
	idx := callbackIndex(ptr)

	var frame [128]uintptr
	frame[callbackIntRegIndex(0)] = 7 // first int arg
	frame[callbackIntRegIndex(1)] = 6 // second int arg

	args := &callbackArgs{
		index:  idx,
		args:   unsafe.Pointer(&frame),
		result: 0,
	}

	callbackWrap(args)

	if args.result != 42 {
		t.Errorf("Expected result 42, got %d", args.result)
	}
}

// Test callback with float64 arguments.
func TestCallback_FloatArgs(t *testing.T) {
	var result float64
	callback := func(a float64, b float64) {
		result = a + b
	}

	ptr := NewCallback(callback)
	idx := callbackIndex(ptr)

	var frame [128]uintptr
	// Float args go in XMM0-7
	f1 := 3.14
	f2 := 2.86
	frame[0] = *(*uintptr)(unsafe.Pointer(&f1)) // XMM0
	frame[1] = *(*uintptr)(unsafe.Pointer(&f2)) // XMM1

	args := &callbackArgs{
		index: idx,
		args:  unsafe.Pointer(&frame),
	}

	callbackWrap(args)

	expected := 6.0
	if result < expected-0.001 || result > expected+0.001 {
		t.Errorf("Expected result ~%.2f, got %.2f", expected, result)
	}
}

// Test callback with float32 arguments.
func TestCallback_Float32Args(t *testing.T) {
	var result float32
	callback := func(a float32, b float32) float32 {
		result = a * b
		return result
	}

	ptr := NewCallback(callback)
	idx := callbackIndex(ptr)

	var frame [128]uintptr
	// Float32 args go in XMM registers (as float64)
	f1 := float64(2.5)
	f2 := float64(4.0)
	frame[0] = *(*uintptr)(unsafe.Pointer(&f1)) // XMM0
	frame[1] = *(*uintptr)(unsafe.Pointer(&f2)) // XMM1

	args := &callbackArgs{
		index:  idx,
		args:   unsafe.Pointer(&frame),
		result: 0,
	}

	callbackWrap(args)

	// Result is float32 stored as float64 bits
	resultF64 := *(*float64)(unsafe.Pointer(&args.result))
	result = float32(resultF64)

	expected := float32(10.0)
	if result < expected-0.001 || result > expected+0.001 {
		t.Errorf("Expected result ~%.2f, got %.2f", expected, result)
	}
}

// Test callback with mixed int and float arguments.
func TestCallback_MixedArgs(t *testing.T) {
	var result float64
	callback := func(count int, multiplier float64) float64 {
		return float64(count) * multiplier
	}

	ptr := NewCallback(callback)
	idx := callbackIndex(ptr)

	var frame [128]uintptr
	// count (int) -> RDI (integer register)
	// multiplier (float64) -> XMM0 (float register)
	frame[callbackIntRegIndex(0)] = 5 // first int arg

	mult := 2.5
	frame[0] = *(*uintptr)(unsafe.Pointer(&mult)) // XMM0 = 2.5

	args := &callbackArgs{
		index:  idx,
		args:   unsafe.Pointer(&frame),
		result: 0,
	}

	callbackWrap(args)

	// Result is float64, stored as bits in args.result
	result = *(*float64)(unsafe.Pointer(&args.result))

	expected := 12.5
	if result < expected-0.001 || result > expected+0.001 {
		t.Errorf("Expected result ~%.2f, got %.2f", expected, result)
	}
}

// Test callback with pointer argument.
func TestCallback_PointerArg(t *testing.T) {
	original := 42
	modified := false

	callback := func(ptr *int) {
		if ptr != nil && *ptr == 42 {
			*ptr = 100
			modified = true
		}
	}

	cbPtr := NewCallback(callback)
	idx := callbackIndex(cbPtr)

	var frame [128]uintptr
	frame[callbackIntRegIndex(0)] = uintptr(unsafe.Pointer(&original)) // first int arg

	args := &callbackArgs{
		index: idx,
		args:  unsafe.Pointer(&frame),
	}

	callbackWrap(args)

	if !modified {
		t.Error("Callback was not called or did not modify value")
	}

	if original != 100 {
		t.Errorf("Expected original value to be 100, got %d", original)
	}
}

// Test callback with boolean argument.
func TestCallback_BoolArg(t *testing.T) {
	var result bool
	callback := func(flag bool) {
		result = flag
	}

	ptr := NewCallback(callback)
	idx := callbackIndex(ptr)

	var frame [128]uintptr
	frame[callbackIntRegIndex(0)] = 1 // first int arg

	args := &callbackArgs{
		index: idx,
		args:  unsafe.Pointer(&frame),
	}

	callbackWrap(args)

	if !result {
		t.Error("Expected result true, got false")
	}
}

// Test callback with boolean return value.
func TestCallback_BoolReturn(t *testing.T) {
	callback := func(x int) bool {
		return x > 0
	}

	ptr := NewCallback(callback)
	idx := callbackIndex(ptr)

	var frame [128]uintptr
	frame[callbackIntRegIndex(0)] = 42 // first int arg

	args := &callbackArgs{
		index:  idx,
		args:   unsafe.Pointer(&frame),
		result: 0,
	}

	callbackWrap(args)

	if args.result != 1 {
		t.Errorf("Expected result 1 (true), got %d", args.result)
	}

	// Test false case
	frame[callbackIntRegIndex(0)] = 0
	args.result = 0
	callbackWrap(args)

	if args.result != 0 {
		t.Errorf("Expected result 0 (false), got %d", args.result)
	}
}

// Test callback with uint types.
func TestCallback_UintTypes(t *testing.T) {
	callback := func(a uint, b uint32, c uint64) uint64 {
		return uint64(a) + uint64(b) + c
	}

	ptr := NewCallback(callback)
	idx := callbackIndex(ptr)

	var frame [128]uintptr
	frame[callbackIntRegIndex(0)] = 10 // first int arg
	frame[callbackIntRegIndex(1)] = 20 // second int arg
	frame[callbackIntRegIndex(2)] = 30 // third int arg

	args := &callbackArgs{
		index:  idx,
		args:   unsafe.Pointer(&frame),
		result: 0,
	}

	callbackWrap(args)

	if args.result != 60 {
		t.Errorf("Expected result 60, got %d", args.result)
	}
}

// Test callback with many arguments (exceeds register count, uses stack).
func TestCallback_StackArgs(t *testing.T) {
	var frame [128]uintptr
	var ptr uintptr
	var idx uintptr
	var expected uintptr
	if runtime.GOARCH == "arm64" {
		callback := func(a, b, c, d, e, f, g, h, i int) int {
			return a + b + c + d + e + f + g + h + i
		}
		ptr = NewCallback(callback)
		idx = callbackIndex(ptr)

		for i := 0; i < 8; i++ {
			frame[callbackIntRegIndex(i)] = uintptr(i + 1)
		}
		frame[callbackStackIndex(0)] = 9
		expected = uintptr(45)
	} else {
		callback := func(a, b, c, d, e, f, g int) int {
			return a + b + c + d + e + f + g
		}
		ptr = NewCallback(callback)
		idx = callbackIndex(ptr)

		for i := 0; i < 6; i++ {
			frame[callbackIntRegIndex(i)] = uintptr(i + 1)
		}
		frame[callbackStackIndex(0)] = 7
		expected = uintptr(28)
	}

	args := &callbackArgs{
		index:  idx,
		args:   unsafe.Pointer(&frame),
		result: 0,
	}

	callbackWrap(args)

	if args.result != expected {
		t.Errorf("Expected result %d, got %d", expected, args.result)
	}
}

// Test callback with mixed float/int stack args.
func TestCallback_StackFloatArgs(t *testing.T) {
	// Use 9 float args to force stack usage (only 8 XMM registers)
	callback := func(f1, f2, f3, f4, f5, f6, f7, f8, f9 float64) float64 {
		return f1 + f2 + f3 + f4 + f5 + f6 + f7 + f8 + f9
	}

	ptr := NewCallback(callback)
	idx := callbackIndex(ptr)

	var frame [128]uintptr
	// First 8 in XMM0-7 (positions 0-7)
	for i := 0; i < 8; i++ {
		val := float64(i + 1)
		frame[i] = *(*uintptr)(unsafe.Pointer(&val))
	}
	val9 := float64(9)
	frame[callbackStackIndex(0)] = *(*uintptr)(unsafe.Pointer(&val9))

	args := &callbackArgs{
		index:  idx,
		args:   unsafe.Pointer(&frame),
		result: 0,
	}

	callbackWrap(args)

	// Sum = 1+2+3+4+5+6+7+8+9 = 45
	resultF64 := *(*float64)(unsafe.Pointer(&args.result))
	expected := 45.0

	if resultF64 < expected-0.001 || resultF64 > expected+0.001 {
		t.Errorf("Expected result ~%.2f, got %.2f", expected, resultF64)
	}
}

// Test multiple callback registrations.
func TestCallback_MultipleRegistrations(t *testing.T) {
	const numCallbacks = 10

	callbacks := make([]uintptr, numCallbacks)
	results := make([]int, numCallbacks)

	// Register multiple callbacks
	for i := 0; i < numCallbacks; i++ {
		idx := i // Capture loop variable
		callback := func(x int) int {
			results[idx] = x * idx
			return x * idx
		}
		callbacks[i] = NewCallback(callback)
	}

	// Verify all callbacks are unique
	seen := make(map[uintptr]bool)
	for i, ptr := range callbacks {
		if seen[ptr] {
			t.Errorf("Callback %d has duplicate pointer", i)
		}
		seen[ptr] = true
	}

	// Invoke each callback
	for i, ptr := range callbacks {
		idx := callbackIndex(ptr)

		var frame [128]uintptr
		frame[callbackIntRegIndex(0)] = 10 // first int arg

		args := &callbackArgs{
			index:  idx,
			args:   unsafe.Pointer(&frame),
			result: 0,
		}

		callbackWrap(args)

		expected := uintptr(10 * i)
		if args.result != expected {
			t.Errorf("Callback %d: expected result %d, got %d", i, expected, args.result)
		}
	}
}

// Test callback limit enforcement.
func TestCallback_LimitEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping limit test in short mode")
	}

	// Save initial callback count
	callbacks.mu.Lock()
	initialCount := callbacks.count
	callbacks.mu.Unlock()

	// Check if we're close to limit already
	if initialCount >= maxCallbacks-10 {
		t.Skip("Callback registry too full for limit test")
	}

	// Test panic on exact limit
	// We won't fill up to limit in test as it's expensive and interferes with other tests
	// Instead, simulate limit condition
	callbacks.mu.Lock()
	savedCount := callbacks.count
	callbacks.count = maxCallbacks // Temporarily set to max
	callbacks.mu.Unlock()

	defer func() {
		// Restore count after test
		callbacks.mu.Lock()
		callbacks.count = savedCount
		callbacks.mu.Unlock()
	}()

	// Next registration should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when exceeding callback limit")
		} else if r != "ffi: callback limit reached (2000 callbacks maximum)" {
			t.Errorf("Wrong panic message: %v", r)
		}
	}()

	callback := func(x int) int {
		return x
	}
	NewCallback(callback)
}

// Test thread safety of callback registration.
func TestCallback_ThreadSafety(t *testing.T) {
	const numGoroutines = 10
	const callbacksPerGoroutine = 5

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Save initial count
	callbacks.mu.Lock()
	initialCount := callbacks.count
	callbacks.mu.Unlock()

	// Register callbacks concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < callbacksPerGoroutine; j++ {
				id := goroutineID*callbacksPerGoroutine + j
				callback := func(x int) int {
					return x + id
				}
				ptr := NewCallback(callback)

				if ptr == 0 {
					t.Errorf("Goroutine %d: NewCallback returned nil", goroutineID)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify correct number of callbacks registered
	callbacks.mu.Lock()
	finalCount := callbacks.count
	callbacks.mu.Unlock()

	expected := initialCount + (numGoroutines * callbacksPerGoroutine)
	if finalCount != expected {
		t.Errorf("Expected %d callbacks, got %d", expected, finalCount)
	}
}

// Test panic on nil callback.
func TestCallback_NilPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil callback")
		} else if r != "ffi: callback function must not be nil" {
			t.Errorf("Wrong panic message: %v", r)
		}
	}()

	NewCallback(nil)
}

// Test panic on non-function callback.
func TestCallback_NonFunctionPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for non-function callback")
		} else if r != "ffi: callback must be a function" {
			t.Errorf("Wrong panic message: %v", r)
		}
	}()

	NewCallback(42)
}

// Test panic on invalid argument type.
func TestCallback_InvalidArgType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid argument type")
		}
	}()

	callback := func(s string) {} // string not supported
	NewCallback(callback)
}

// Test panic on invalid return type.
func TestCallback_InvalidReturnType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid return type")
		}
	}()

	callback := func() string { return "" } // string not supported
	NewCallback(callback)
}

// Test panic on multiple return values.
func TestCallback_MultipleReturnsPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for multiple return values")
		} else if r != "ffi: callbacks can only return zero or one value" {
			t.Errorf("Wrong panic message: %v", r)
		}
	}()

	callback := func() (int, int) { return 1, 2 }
	NewCallback(callback)
}

// Test memory safety: ensure callbacks aren't garbage collected.
func TestCallback_NoGarbageCollection(t *testing.T) {
	var result int
	callback := func(x int) {
		result = x
	}

	ptr := NewCallback(callback)

	// Force garbage collection
	runtime.GC()
	runtime.GC()

	// Callback should still be callable
	idx := callbackIndex(ptr)

	var frame [128]uintptr
	frame[callbackIntRegIndex(0)] = 99

	args := &callbackArgs{
		index: idx,
		args:  unsafe.Pointer(&frame),
	}

	callbackWrap(args)

	if result != 99 {
		t.Errorf("Expected result 99 after GC, got %d", result)
	}
}

// Benchmark callback registration.
func BenchmarkNewCallback(b *testing.B) {
	callback := func(x int) int {
		return x * 2
	}

	// Save initial count
	callbacks.mu.Lock()
	initialCount := callbacks.count
	callbacks.mu.Unlock()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Check if we're approaching the limit
		callbacks.mu.Lock()
		if callbacks.count >= maxCallbacks {
			callbacks.count = initialCount // Reset for benchmark
		}
		callbacks.mu.Unlock()

		NewCallback(callback)
	}
}

// Benchmark callback invocation.
func BenchmarkCallbackInvoke(b *testing.B) {
	callback := func(a int, b int) int {
		return a + b
	}

	ptr := NewCallback(callback)
	idx := callbackIndex(ptr)

	var frame [128]uintptr
	frame[callbackIntRegIndex(0)] = 10
	frame[callbackIntRegIndex(1)] = 20

	args := &callbackArgs{
		index:  idx,
		args:   unsafe.Pointer(&frame),
		result: 0,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		callbackWrap(args)
	}
}

// Benchmark callback with float arguments.
func BenchmarkCallbackFloat(b *testing.B) {
	callback := func(a float64, b float64) float64 {
		return a * b
	}

	ptr := NewCallback(callback)
	idx := callbackIndex(ptr)

	var frame [128]uintptr
	f1 := 3.14
	f2 := 2.0
	frame[0] = *(*uintptr)(unsafe.Pointer(&f1))
	frame[1] = *(*uintptr)(unsafe.Pointer(&f2))

	args := &callbackArgs{
		index:  idx,
		args:   unsafe.Pointer(&frame),
		result: 0,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		callbackWrap(args)
	}
}
