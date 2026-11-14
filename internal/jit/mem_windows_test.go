// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build windows && amd64

package jit

import (
	"syscall"
	"testing"
	"unsafe"
)

// TestUnwindInfo unwinds a bridge frame with RtlVirtualUnwind, as the
// exception dispatcher and debuggers do, and checks that it finds the
// caller: RBP and the return address from the frame RBP points to, and RSP
// just above them.
func TestUnwindInfo(t *testing.T) {
	code := append(append([]byte{}, prologue...),
		0x90, 0x90, 0x90, 0x90, // body
		0x48, 0x8d, 0x65, 0x00, 0x5d, 0xc3) // lea rsp, [rbp]; pop rbp; ret
	c := Install(code)
	if c == nil {
		t.Skip("call bridges are disabled")
	}
	if !c.chunk.seal() {
		t.Fatal("seal failed")
	}
	pc := c.entry + uintptr(len(prologue)) + 1

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	var base uint64
	entry, _, _ := kernel32.NewProc("RtlLookupFunctionEntry").Call(pc, uintptr(unsafe.Pointer(&base)), 0)
	if entry == 0 {
		t.Fatalf("RtlLookupFunctionEntry(%#x) found no unwind information", pc)
	}

	// A CONTEXT is 1232 bytes, 16-byte aligned; RSP, RBP and RIP are at
	// 0x98, 0xa0 and 0xf8.
	buf := make([]byte, 1232+16)
	ctx := unsafe.Pointer(&buf[16-uintptr(unsafe.Pointer(&buf[0]))%16])
	reg := func(off uintptr) *uint64 { return (*uint64)(unsafe.Add(ctx, off)) }
	frame := [2]uint64{0x1111, 0x2222} // saved RBP, return address
	*reg(0x98) = uint64(uintptr(unsafe.Pointer(&frame[0]))) - 0x40
	*reg(0xa0) = uint64(uintptr(unsafe.Pointer(&frame[0])))
	*reg(0xf8) = uint64(pc)
	var handlerData, establisher uint64
	kernel32.NewProc("RtlVirtualUnwind").Call(0, uintptr(base), pc, entry, uintptr(ctx),
		uintptr(unsafe.Pointer(&handlerData)), uintptr(unsafe.Pointer(&establisher)), 0)

	if rip, rbp, rsp := *reg(0xf8), *reg(0xa0), *reg(0x98); rip != 0x2222 || rbp != 0x1111 ||
		rsp != uint64(uintptr(unsafe.Pointer(&frame[1])))+8 {
		t.Errorf("unwound to RIP %#x RBP %#x RSP %#x, want 0x2222 0x1111 %p+16", rip, rbp, rsp, &frame[0])
	}
}
