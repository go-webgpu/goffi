// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build windows && amd64

package jit

import (
	"bytes"
	"syscall"
	"unsafe"
)

const supported = true

var (
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procVirtualAlloc          = kernel32.NewProc("VirtualAlloc")
	procVirtualProtect        = kernel32.NewProc("VirtualProtect")
	procFlushInstructionCache = kernel32.NewProc("FlushInstructionCache")
	procRtlAddFunctionTable   = kernel32.NewProc("RtlAddFunctionTable")
)

const (
	memCommit       = 0x1000
	memReserve      = 0x2000
	pageReadWrite   = 0x04
	pageExecuteRead = 0x20
)

// Windows finds the handler for an exception raised in C code by unwinding
// the stack, which needs unwind information for every frame, bridges
// included. Every bridge starts with the same prologue and keeps RBP as its
// frame pointer, so one UNWIND_INFO, stored at unwindOffset in each chunk,
// describes all of them: version 1, a 4-byte prologue, frame register RBP
// at offset 0, and two unwind codes, UWOP_SET_FPREG after `mov rbp, rsp`
// and UWOP_PUSH_NONVOL RBP after `push rbp`.
var (
	prologue   = []byte{0x55, 0x48, 0x89, 0xe5} // push rbp; mov rbp, rsp
	unwindInfo = []byte{0x01, 0x04, 0x02, 0x05, 0x04, 0x03, 0x01, 0x50}
)

const (
	unwindOffset = 4
	chunkHeader  = 16
)

// runtimeFunction is RUNTIME_FUNCTION: offsets from the chunk start.
type runtimeFunction struct {
	begin, end, unwind uint32
}

// tables keeps the registered function tables alive: Windows keeps a
// pointer to each one.
var tables [][]runtimeFunction

func validCode(code []byte) bool { return bytes.HasPrefix(code, prologue) }

func allocExec(n int) ([]byte, error) {
	addr, _, err := procVirtualAlloc.Call(0, uintptr(n), memCommit|memReserve, pageReadWrite)
	if addr == 0 {
		return nil, err
	}
	return unsafe.Slice(*(**byte)(unsafe.Pointer(&addr)), n), nil
}

func sealExec(k *chunk) error {
	copy(k.mem[unwindOffset:], unwindInfo)
	base := uintptr(unsafe.Pointer(unsafe.SliceData(k.mem)))
	size := uintptr(len(k.mem))
	var old uint32
	if ok, _, err := procVirtualProtect.Call(base, size, pageExecuteRead, uintptr(unsafe.Pointer(&old))); ok == 0 {
		return err
	}
	process, err := syscall.GetCurrentProcess()
	if err != nil {
		return err
	}
	if ok, _, err := procFlushInstructionCache.Call(uintptr(process), base, size); ok == 0 {
		return err
	}
	table := make([]runtimeFunction, len(k.spans))
	for i, s := range k.spans {
		table[i] = runtimeFunction{uint32(s[0]), uint32(s[1]), unwindOffset}
	}
	if ok, _, err := procRtlAddFunctionTable.Call(uintptr(unsafe.Pointer(&table[0])), uintptr(len(table)), base); ok&0xff == 0 {
		return err
	}
	tables = append(tables, table)
	return nil
}
