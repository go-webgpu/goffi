//go:build windows

package ffi

import (
	"sync"
	"testing"
)

// Test basic callback registration on Windows.
func TestNewCallback_BasicRegistration(t *testing.T) {
	callback := func(a, b, c, d uintptr) uintptr {
		return a + b + c + d
	}

	ptr := NewCallback(callback)

	if ptr == 0 {
		t.Fatal("NewCallback returned nil pointer")
	}
}

// Test callback with multiple arguments.
func TestNewCallback_MultipleArgs(t *testing.T) {
	callback := func(a, b, c, d, e uintptr) uintptr {
		return a + b + c + d + e
	}

	ptr := NewCallback(callback)
	if ptr == 0 {
		t.Fatal("NewCallback returned nil pointer")
	}
}

// Test callback count tracking.
func TestCallbackCount(t *testing.T) {
	initialCount := CallbackCount()

	_ = NewCallback(func(uintptr) uintptr { return 0 })
	_ = NewCallback(func(uintptr) uintptr { return 0 })
	_ = NewCallback(func(uintptr) uintptr { return 0 })

	finalCount := CallbackCount()

	if finalCount != initialCount+3 {
		t.Errorf("Expected callback count to increase by 3, got %d -> %d", initialCount, finalCount)
	}
}

// Test nil callback panic.
func TestNewCallback_NilPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil callback")
		}
	}()
	NewCallback(nil)
}

// Test non-function callback panic.
func TestNewCallback_NonFunctionPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for non-function callback")
		}
	}()
	NewCallback("not a function")
}

// Test float argument panic (not supported on Windows syscall.NewCallback).
func TestNewCallback_FloatArgPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for float argument")
		}
	}()
	NewCallback(func(f float64) {})
}

// Test float return panic (not supported on Windows syscall.NewCallback).
func TestNewCallback_FloatReturnPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for float return")
		}
	}()
	NewCallback(func() float64 { return 0 })
}

// Test multiple return values panic.
func TestNewCallback_MultipleReturnsPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for multiple return values")
		}
	}()
	NewCallback(func() (int, int) { return 0, 0 })
}

// Test concurrent callback registration.
func TestNewCallback_Concurrent(t *testing.T) {
	const numGoroutines = 10
	const callbacksPerGoroutine = 5

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make(chan error, numGoroutines*callbacksPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < callbacksPerGoroutine; j++ {
				ptr := NewCallback(func(x uintptr) uintptr { return x })
				if ptr == 0 {
					errors <- nil
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Error("Callback registration failed")
		}
	}
}

// Test various supported argument types.
// Note: On Windows, syscall.NewCallback requires uintptr-sized arguments AND exactly one return value.
func TestNewCallback_SupportedTypes(t *testing.T) {
	tests := []struct {
		name string
		fn   any
	}{
		// Windows syscall.NewCallback requires uintptr-sized types and exactly one return
		{"uintptr args", func(a, b uintptr) uintptr { return a + b }},
		{"no args", func() uintptr { return 42 }},
		{"single arg", func(a uintptr) uintptr { return a }},
		{"five args", func(a, b, c, d, e uintptr) uintptr { return a + b + c + d + e }},
		{"int64 args", func(a, b int64) int64 { return a + b }},
		{"uint64 args", func(a, b uint64) uint64 { return a + b }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptr := NewCallback(tt.fn)
			if ptr == 0 {
				t.Errorf("NewCallback returned nil for %s", tt.name)
			}
		})
	}
}

// Test that void return panics (Windows requires exactly one return value).
func TestNewCallback_VoidReturnPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for void return")
		}
	}()
	NewCallback(func(a, b uintptr) {})
}

// Test that non-uintptr sized types panic (Windows limitation).
func TestNewCallback_NonUintptrTypePanic(t *testing.T) {
	tests := []struct {
		name string
		fn   any
	}{
		{"int8 args", func(a int8) int8 { return a }},
		{"int16 args", func(a int16) int16 { return a }},
		{"int32 args", func(a int32) int32 { return a }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected panic for %s", tt.name)
				}
			}()
			NewCallback(tt.fn)
		})
	}
}
