// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

// Package jit runs call bridges: machine code compiled for one call
// signature that loads each argument from avalue into its register or stack
// slot, calls the C function, stores the result in rvalue and returns errno.
// The arch packages generate the code; this package owns the executable
// memory it runs from and the call into it.
//
// A bridge is entered through runtime.cgocall with a pointer to a frame:
//
//	offset 0   fn       the C function
//	offset 8   avalue   &avalue[0]
//	offset 16  rvalue   result memory, or nil
//	offset 24  errnoFn  __errno_location or __error, or 0 to skip errno
//
// It returns the captured errno as a 32-bit value. The frame lives on the
// goroutine stack, which a Go callback made during the C call may move, so
// a bridge reads every frame field before it calls the C function.
package jit

import (
	"os"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Code is a bridge in executable memory.
type Code struct {
	entry uintptr
	chunk *chunk
}

// A chunk is a block of memory that bridges are written into. It is
// writable until the first call through any of its bridges, which seals it:
// from then on it is read-execute only, and new bridges go to a new chunk.
// Bridges prepared together, as when a library prepares all of its
// functions at load time, share one chunk.
type chunk struct {
	mem   []byte
	used  int
	state atomic.Int32
	spans [][2]int // start and end offset of each bridge
}

const (
	open int32 = iota
	sealed
	failed
)

var (
	mu       sync.Mutex
	current  *chunk
	installs = map[string]*Code{}
	enabled  = supported && os.Getenv("GOFFI_JIT") != "0"
)

// Install returns code in executable memory, shared with identical code
// installed before. It returns nil when bridges are disabled or the
// platform cannot run them; the caller then uses its generic call path.
func Install(code []byte) *Code {
	mu.Lock()
	defer mu.Unlock()
	if !enabled || !validCode(code) {
		return nil
	}
	if c, ok := installs[string(code)]; ok {
		return c
	}
	size := (len(code) + 15) &^ 15
	k := current
	if k == nil || k.state.Load() != open || k.used+size > len(k.mem) {
		mem, err := allocExec(max(size+chunkHeader, os.Getpagesize()))
		if err != nil {
			return nil
		}
		k = &chunk{mem: mem, used: chunkHeader}
		current = k
	}
	off := k.used
	copy(k.mem[off:], code)
	k.used += size
	k.spans = append(k.spans, [2]int{off, off + len(code)})
	c := &Code{entry: uintptr(unsafe.Pointer(&k.mem[off])), chunk: k}
	installs[string(code)] = c
	return c
}

// seal makes k executable and reports whether it is.
func (k *chunk) seal() bool {
	mu.Lock()
	defer mu.Unlock()
	if k.state.Load() == open {
		if err := sealExec(k); err != nil {
			// The platform refuses executable memory (macOS hardened
			// runtime, an SELinux execmem policy): stop compiling bridges.
			k.state.Store(failed)
			enabled = false
		} else {
			k.state.Store(sealed)
		}
	}
	return k.state.Load() == sealed
}

type frame struct {
	fn      unsafe.Pointer
	avalue  *unsafe.Pointer
	rvalue  unsafe.Pointer
	errnoFn uintptr
}

//go:linkname runtime_cgocall runtime.cgocall
//go:noescape
func runtime_cgocall(fn uintptr, arg unsafe.Pointer) int32

// Call calls fn through c and returns the errno captured right after it. It
// reports false without calling when c cannot run: an argument pointer is
// nil, which the generic call path turns into a Go panic, or c's memory
// could not be made executable.
func Call(c *Code, fn, rvalue unsafe.Pointer, avalue []unsafe.Pointer, errnoFn uintptr) (uintptr, bool) {
	for _, p := range avalue {
		if p == nil {
			return 0, false
		}
	}
	switch c.chunk.state.Load() {
	case sealed:
	case open:
		if !c.chunk.seal() {
			return 0, false
		}
	default:
		return 0, false
	}
	f := frame{fn: fn, avalue: unsafe.SliceData(avalue), rvalue: rvalue, errnoFn: errnoFn}
	errno := runtime_cgocall(c.entry, unsafe.Pointer(&f))
	return uintptr(uint32(errno)), true
}
