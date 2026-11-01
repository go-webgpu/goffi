//go:build amd64 && (linux || darwin || freebsd)

// Unix implementation using System V AMD64 ABI (Linux, macOS, FreeBSD, etc.)
// The argument layout is layout_sysv.go; this file places the values.

package amd64

import (
	"fmt"
	"math"
	"runtime"
	"unsafe"

	gosyscall "github.com/go-webgpu/goffi/internal/syscall"
	"github.com/go-webgpu/goffi/types"
)

// loadWord returns the n (1-8) bytes at p in the low bytes of a word. It reads
// exactly n bytes: p points into the caller's value, which may end there.
func loadWord(p unsafe.Pointer, n uintptr) uintptr {
	var w uintptr
	copy((*[8]byte)(unsafe.Pointer(&w))[:n], unsafe.Slice((*byte)(p), n))
	return w
}

// smallStackWords is how many stack words a call places without allocating.
const smallStackWords = 16

func (i *Implementation) Execute(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
	errnoFn uintptr,
) (cerrno uintptr, err error) {
	// The stack area is sized before any value is placed, so an argument list
	// that does not fit is an error, never a write past the area. A
	// CallInterface filled in by hand reaches this point without the check in
	// PrepareCallInterface.
	stackBytes := sysvStackBytes(cif)
	if stackBytes > MaxStackBytes {
		return 0, fmt.Errorf("goffi: arguments need %d bytes of stack, the amd64 call path supports %d",
			stackBytes, MaxStackBytes)
	}
	var (
		gpr   [sysvGPRs]uintptr
		sse   [sysvSSEs]float64
		small [smallStackWords]uintptr
		alloc SysVAllocator
	)
	var stack []uintptr
	if n := int(stackBytes / 8); n > len(small) {
		stack = make([]uintptr, n)
	} else {
		stack = small[:n]
	}
	setSSE := func(reg int, bits uintptr) {
		*(*uintptr)(unsafe.Pointer(&sse[reg])) = bits
	}

	// A struct result over 16 bytes is returned through caller memory whose
	// address is the hidden first argument in RDI; the callee writes the
	// result there directly.
	sret := sysvRetInMemory(cif)
	if sret {
		gpr[alloc.GPR()] = uintptr(rvalue)
	}

	// ffi.CallFunctionContext has checked that avalue holds a value for every
	// argument type.
	for idx, argType := range cif.ArgTypes {
		val := avalue[idx]
		p := alloc.place(argType)

		if argType.Kind == types.StructType {
			if p.onStack {
				sz := argType.Size
				for off := uintptr(0); off < sz; off += 8 {
					stack[p.word+int(off/8)] = loadWord(unsafe.Add(val, off), min(8, sz-off))
				}
				continue
			}
			for k := range p.n {
				off := uintptr(k) * 8
				w := loadWord(unsafe.Add(val, off), min(8, argType.Size-off))
				if p.sse[k] {
					setSSE(p.reg[k], w)
				} else {
					gpr[p.reg[k]] = w
				}
			}
			continue
		}

		w := scalarWord(argType, val)
		switch {
		case p.onStack:
			stack[p.word] = w
		case p.sse[0]:
			setSSE(p.reg[0], w)
		default:
			gpr[p.reg[0]] = w
		}
	}

	// Call via syscall; errnoFn is non-zero on Unix, 0 on Windows.
	// When errnoFn is 0, the assembly skips errno capture (TESTQ/JZ).
	ret, r2, fret, fret2, capturedErrno := gosyscall.CallNFloatErrno(uintptr(fn), gpr, sse, stack, errnoFn)

	runtime.KeepAlive(avalue)
	runtime.KeepAlive(rvalue)

	// If sret, the callee wrote directly into rvalue — no further copy needed.
	if sret {
		return capturedErrno, nil
	}

	// Handle return value based on type
	retVal := uint64(ret)

	// For float returns, use the float value from XMM0
	if cif.ReturnType.Kind == types.FloatType || cif.ReturnType.Kind == types.DoubleType {
		retVal = *(*uint64)(unsafe.Pointer(&fret))
	}

	return capturedErrno, i.handleReturn(cif, rvalue, retVal, uint64(r2), fret, fret2)
}

// scalarWord returns the register or stack word for a non-struct argument.
// Narrow integers are sign- or zero-extended to the full register, as libffi
// does (src/x86/ffi64.c, ffi_call_int). The psABI leaves the upper bits
// unspecified, but clang-built callees rely on the caller extending char and
// short to 32 bits: Apple clang compiles `int64_t f(int8_t a) { return a; }`
// to `movslq %edi, %rax`. A float keeps its exact 32-bit pattern in the low
// bits: widening it to float64 would corrupt what the callee reads.
func scalarWord(t *types.TypeDescriptor, p unsafe.Pointer) uintptr {
	switch t.Kind {
	case types.FloatType:
		return uintptr(math.Float32bits(*(*float32)(p)))
	case types.DoubleType, types.PointerType, types.SInt64Type, types.UInt64Type:
		return *(*uintptr)(p)
	case types.SInt8Type:
		return uintptr(int64(*(*int8)(p)))
	case types.UInt8Type:
		return uintptr(*(*uint8)(p))
	case types.SInt16Type:
		return uintptr(int64(*(*int16)(p)))
	case types.UInt16Type:
		return uintptr(*(*uint16)(p))
	case types.SInt32Type, types.IntType:
		return uintptr(int64(*(*int32)(p)))
	case types.UInt32Type:
		return uintptr(*(*uint32)(p))
	default:
		// For unknown/composite types, pass as pointer-to-value
		return uintptr(p)
	}
}
