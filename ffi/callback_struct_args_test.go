//go:build (linux || darwin || freebsd) && (amd64 || arm64)

package ffi

import (
	"testing"
	"unsafe"
)

func testCallbackStruct[T comparable](
	t *testing.T,
	// Argument frame (System V AMD64 ABI)
	// Layout: [XMM0-7][RDI,RSI,RDX,RCX,R8,R9][stack...]
	frame [128]uintptr,
	// A struct that is the expected callback argument
	expected T,
) {
	t.Helper()

	var arg T
	callback := func(s T) { arg = s }

	ptr := NewCallback(callback)
	if ptr == 0 {
		t.Fatal("NewCallback returned nil pointer")
	}

	idx := callbackIndex(ptr)

	args := &callbackArgs{
		index: idx,
		args:  unsafe.Pointer(&frame),
	}

	callbackWrap(args)

	if arg != expected {
		t.Errorf("Expected result %#v, got %#v", expected, arg)
	}
}

func TestCallbackStructEmpty(t *testing.T) {
	type struct_ struct{}
	testCallbackStruct(t, [128]uintptr{}, struct_{})
}

func TestCallbackStructSingleFloat(t *testing.T) {
	type struct_ struct{ a float32 }
	expected := struct_{a: 3.14}

	testCallbackStruct(t, [128]uintptr{
		*(*uintptr)(unsafe.Pointer(&expected.a)), // XMM0
	}, expected)
}

func TestCallbackStruct8BTwoFloat32s(t *testing.T) {
	type struct_ struct {
		a float32
		b float32
	}
	expected := struct_{a: 3.14, b: 2.86}

	// Pack 2 floats in to a 64 bit area.
	var floats [8]byte
	*(*float32)(unsafe.Pointer(&floats[0])) = expected.a
	*(*float32)(unsafe.Pointer(&floats[4])) = expected.b
	frame := [128]uintptr{}
	frame[0] = *(*uintptr)(unsafe.Pointer(&floats)) // XMM0
	testCallbackStruct(t, frame, expected)
}

func TestCallbackStruct8BFloat64(t *testing.T) {
	type struct_ struct {
		a float64
	}
	expected := struct_{a: 3.14}

	frame := [128]uintptr{}
	frame[0] = *(*uintptr)(unsafe.Pointer(&expected.a)) // XMM0
	testCallbackStruct(t, frame, expected)
}

func TestCallbackStruct8BMixedIntegers3(t *testing.T) {
	type struct_ struct {
		a byte
		b int16
	}
	expected := struct_{a: 0x10, b: 0x2000}

	frame := [128]uintptr{}
	frame[callbackIntRegIndex(0)] = 0x20000010 // first int arg
	testCallbackStruct(t, frame, expected)
}

func TestCallbackStruct6BMixedIntegers(t *testing.T) {
	type struct_ struct {
		a byte
		b int16
		c byte
	}
	expected := struct_{a: 0x10, b: 0x2000, c: 0x30}

	frame := [128]uintptr{}
	frame[callbackIntRegIndex(0)] = 0x3020000010 // first int arg
	testCallbackStruct(t, frame, expected)
}

func TestCallbackStruct8BInt64(t *testing.T) {
	type struct_ struct {
		a int64
	}
	expected := struct_{a: -10}

	frame := [128]uintptr{}
	frame[callbackIntRegIndex(0)] = uintptr(expected.a) // first int arg
	testCallbackStruct(t, frame, expected)
}

func TestCallbackStruct16BFloatFloat(t *testing.T) {
	type struct_ struct {
		a float64
		b float64
	}
	expected := struct_{a: 3.14, b: 2.86}

	frame := [128]uintptr{}
	frame[0] = *(*uintptr)(unsafe.Pointer(&expected.a)) // XMM0
	frame[1] = *(*uintptr)(unsafe.Pointer(&expected.b)) // XMM1
	testCallbackStruct(t, frame, expected)
}

func TestCallbackStruct16BIntInt(t *testing.T) {
	type struct_ struct {
		a int64
		b int64
	}
	expected := struct_{a: 10, b: 20}

	frame := [128]uintptr{}
	frame[callbackIntRegIndex(0)] = uintptr(expected.a) // first int arg
	frame[callbackIntRegIndex(1)] = uintptr(expected.b) // second int arg
	testCallbackStruct(t, frame, expected)
}

func TestCallbackStruct16BFloatInt(t *testing.T) {
	type struct_ struct {
		a float64
		b int64
	}
	expected := struct_{a: 3.14, b: 20}

	frame := [128]uintptr{}
	frame[0] = *(*uintptr)(unsafe.Pointer(&expected.a)) // XMM0
	frame[callbackIntRegIndex(0)] = uintptr(expected.b) // first int arg
	testCallbackStruct(t, frame, expected)
}

func TestCallbackStruct16BIntFloat(t *testing.T) {
	type struct_ struct {
		a int64
		b float64
	}
	expected := struct_{a: 20, b: 3.14}

	frame := [128]uintptr{}
	frame[callbackIntRegIndex(0)] = uintptr(expected.a) // first int arg
	frame[0] = *(*uintptr)(unsafe.Pointer(&expected.b)) // XMM0
	testCallbackStruct(t, frame, expected)
}

func TestCallbackStruct17B(t *testing.T) {
	type struct_ struct {
		a int64
		b int64
		c int8 // partial chunk - 1 byte
	}
	expected := struct_{a: 10, b: 20, c: 30}

	frame := [128]uintptr{}
	frame[callbackStackIndex(0)] = uintptr(expected.a)
	frame[callbackStackIndex(1)] = uintptr(expected.b)
	frame[callbackStackIndex(2)] = uintptr(expected.c)
	testCallbackStruct(t, frame, expected)
}

func TestCallbackStruct18B(t *testing.T) {
	type struct_ struct {
		a int64
		b int64
		c int16 // partial chunk - 2 bytes
	}
	expected := struct_{a: 10, b: 20, c: 30}

	frame := [128]uintptr{}
	frame[callbackStackIndex(0)] = uintptr(expected.a)
	frame[callbackStackIndex(1)] = uintptr(expected.b)
	frame[callbackStackIndex(2)] = uintptr(expected.c)
	testCallbackStruct(t, frame, expected)
}

func TestCallbackStruct20B(t *testing.T) {
	type struct_ struct {
		a int64
		b int64
		c int32 // partial chunk - 4 bytes
	}
	expected := struct_{a: 10, b: 20, c: 30}

	frame := [128]uintptr{}
	frame[callbackStackIndex(0)] = uintptr(expected.a)
	frame[callbackStackIndex(1)] = uintptr(expected.b)
	frame[callbackStackIndex(2)] = uintptr(expected.c)
	testCallbackStruct(t, frame, expected)
}
