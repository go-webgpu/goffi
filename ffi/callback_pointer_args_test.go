//go:build ((linux && !android) || darwin || freebsd) && amd64 && !goffi_static

package ffi

import (
	"runtime"
	"testing"
	"unsafe"
)

// These tests drive callbackWrap directly with a hand-built System V AMD64
// argument frame, exactly as the assembly trampoline would. They cover the
// pointer, unsafe.Pointer, and stack-spilled argument decode paths and the
// pointer return path, which the higher-level callback tests do not reach.
//
// Frame layout: [XMM0-7: 8 slots][RDI,RSI,RDX,RCX,R8,R9: 6 slots][stack...].
const (
	cbFloatRegs = 8                       // XMM0-XMM7
	cbIntRegs   = 6                       // RDI, RSI, RDX, RCX, R8, R9
	cbStackBase = cbFloatRegs + cbIntRegs // first stack argument slot
)

// invokeCallback registers cb, drives callbackWrap with frame, and returns the
// marshaled return value (callbackArgs.result).
func invokeCallback(t *testing.T, cb any, frame *[128]uintptr) uintptr {
	t.Helper()
	ptr := NewCallback(cb)
	if ptr == 0 {
		t.Fatal("NewCallback returned nil pointer")
	}
	a := &callbackArgs{index: callbackIndex(ptr), args: unsafe.Pointer(frame)}
	callbackWrap(a)
	return a.result
}

// fillIntRegs sets the six integer argument registers to distinct nonzero
// values so that the next argument is forced onto the stack.
func fillIntRegs(frame *[128]uintptr) {
	for i := 0; i < cbIntRegs; i++ {
		frame[cbFloatRegs+i] = uintptr(int64(i + 1))
	}
}

func TestCallbackPointerArgInRegister(t *testing.T) {
	want := new(int64)
	*want = 0x0BADC0DE
	var got *int64

	var frame [128]uintptr
	frame[cbFloatRegs] = uintptr(unsafe.Pointer(want)) // RDI
	invokeCallback(t, func(p *int64) { got = p }, &frame)
	runtime.KeepAlive(want)

	if got != want {
		t.Fatalf("pointer arg: got %p, want %p", got, want)
	}
	if *got != *want {
		t.Errorf("deref: got %#x, want %#x", *got, *want)
	}
}

func TestCallbackPointerArgOnStack(t *testing.T) {
	want := new(int64)
	*want = 0x0FACADE
	var got *int64

	var frame [128]uintptr
	fillIntRegs(&frame)
	frame[cbStackBase] = uintptr(unsafe.Pointer(want)) // seventh arg spills to stack
	invokeCallback(t, func(_, _, _, _, _, _ int64, p *int64) { got = p }, &frame)
	runtime.KeepAlive(want)

	if got != want {
		t.Fatalf("stack pointer arg: got %p, want %p", got, want)
	}
	if *got != *want {
		t.Errorf("deref: got %#x, want %#x", *got, *want)
	}
}

func TestCallbackUnsafePointerArgInRegister(t *testing.T) {
	want := new(int64)
	var got unsafe.Pointer

	var frame [128]uintptr
	frame[cbFloatRegs] = uintptr(unsafe.Pointer(want)) // RDI
	invokeCallback(t, func(p unsafe.Pointer) { got = p }, &frame)
	runtime.KeepAlive(want)

	if got != unsafe.Pointer(want) {
		t.Fatalf("unsafe.Pointer arg: got %p, want %p", got, unsafe.Pointer(want))
	}
}

func TestCallbackUnsafePointerArgOnStack(t *testing.T) {
	want := new(int64)
	var got unsafe.Pointer

	var frame [128]uintptr
	fillIntRegs(&frame)
	frame[cbStackBase] = uintptr(unsafe.Pointer(want)) // seventh arg spills to stack
	invokeCallback(t, func(_, _, _, _, _, _ int64, p unsafe.Pointer) { got = p }, &frame)
	runtime.KeepAlive(want)

	if got != unsafe.Pointer(want) {
		t.Fatalf("stack unsafe.Pointer arg: got %p, want %p", got, unsafe.Pointer(want))
	}
}

func TestCallbackBoolArgOnStack(t *testing.T) {
	var got bool

	var frame [128]uintptr
	fillIntRegs(&frame)
	frame[cbStackBase] = 1 // seventh arg (bool) spills to stack
	invokeCallback(t, func(_, _, _, _, _, _ int64, b bool) { got = b }, &frame)

	if !got {
		t.Error("stack bool arg: got false, want true")
	}
}

func TestCallbackPointerReturn(t *testing.T) {
	want := new(int64)

	var frame [128]uintptr
	res := invokeCallback(t, func() *int64 { return want }, &frame)
	runtime.KeepAlive(want)

	if res != uintptr(unsafe.Pointer(want)) {
		t.Fatalf("pointer return: got %#x, want %#x", res, uintptr(unsafe.Pointer(want)))
	}
}
