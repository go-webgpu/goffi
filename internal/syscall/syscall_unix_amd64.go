//go:build (linux || darwin || freebsd) && amd64

// System V AMD64 ABI syscall implementation (Linux, macOS, FreeBSD, etc.)
// This calling convention is IDENTICAL on all Unix-like systems.
package syscall

import (
	"structs"
	"sync"
	"unsafe"
)

//go:linkname runtime_cgocall runtime.cgocall
func runtime_cgocall(fn uintptr, arg unsafe.Pointer) int32

// syscallArgsPool recycles the per-call argument/return block. The block must
// live in non-moving memory (see CallNFloatErrno): pooling keeps it
// heap-resident and reused, so the move-safe path costs no per-call allocation
// in steady state.
var syscallArgsPool = sync.Pool{New: func() any { return new(syscallArgs) }}

// syscallArgs matches the layout expected by syscallN assembly.
//
// Layout (offsets in bytes):
//
//	fn:      0
//	a1-a6:   8-48    (RDI, RSI, RDX, RCX, R8, R9)
//	stack:   56      (address of the first stack word)
//	nstack:  64      (number of 8-byte stack words)
//	f1-f8:   72-128  (XMM0-XMM7 as bit patterns)
//	r1:      136     (RAX return)
//	r2:      144     (RDX return, used for 9-16 byte struct returns)
//	errno:   152     (captured C errno value; 0 if errnoFn == 0)
//	errnoFn: 160     (address of __errno_location/__error; 0 = skip errno capture)
type syscallArgs struct {
	_                      structs.HostLayout
	fn                     uintptr
	a1, a2, a3, a4, a5, a6 uintptr
	stack                  uintptr
	nstack                 uintptr
	f1, f2, f3, f4         uintptr
	f5, f6, f7, f8         uintptr
	r1, r2                 uintptr
	errno                  uintptr
	errnoFn                uintptr

	// words backs stack: the stack arguments are copied here, into the pooled
	// heap block, so syscallN reads them from memory that does not move. It
	// is not part of the layout syscallN sees.
	words []uintptr
}

// syscallN is implemented in syscall_unix_amd64.s
//
//nolint:unused // Called from assembly (syscall_unix_amd64.s)
func syscallN(args unsafe.Pointer)

// syscallNABI0 is the ABI0 entry point for syscallN
var syscallNABI0 uintptr

// CallNFloat is CallNFloatErrno without errno capture.
func CallNFloat(fn uintptr, gpr [6]uintptr, sse [8]float64, stack []uintptr) (r1 uintptr, r2 uintptr, f1 float64, f2 float64) {
	r1, r2, f1, f2, _ = CallNFloatErrno(fn, gpr, sse, stack, 0)
	return
}

// CallNFloatErrno calls a C function with 6 integer register arguments, 8 SSE
// register arguments and len(stack) 8-byte stack words, and captures the C
// errno value set by the called function.
//
// gpr:   RDI, RSI, RDX, RCX, R8, R9
// sse:   XMM0-XMM7 as float64 bit patterns
// stack: the stack words in order: stack[0] is at the stack pointer at the
// call. syscallN reserves exactly this many words on the system stack, so the
// caller bounds len(stack) (amd64.MaxStackBytes).
//
// The errno function address (errnoFn) must be the address of __errno_location
// (Linux/FreeBSD) or __error (macOS), obtained from ErrnoFnAddr(). When errnoFn
// is 0, errno capture is skipped and the returned errno value is always 0. The
// errno is read inside the assembly trampoline immediately after the C function
// returns, before the Go runtime can migrate the goroutine to a different OS
// thread. This is the only safe window for errno capture.
//
// Returns:
//   - r1: RAX integer return value
//   - r2: RDX second integer return value (9-16 byte struct returns)
//   - f1: XMM0 float return value (bit pattern)
//   - f2: XMM1 second float return value — for {SSE, SSE} 9-16B struct returns (e.g. NSPoint)
func CallNFloatErrno(fn uintptr, gpr [6]uintptr, sse [8]float64, stack []uintptr, errnoFn uintptr) (r1 uintptr, r2 uintptr, f1 float64, f2 float64, cerrno uintptr) {
	// args holds the call arguments, the stack words and the C function's
	// return values. syscallN runs on g0 and writes the return values back
	// into args after the call returns. That call can run a callback that
	// re-enters Go and grows this goroutine's stack, which moves it. If args
	// lived on the stack it would move too, syscallN's write would land at the
	// old address, and the first such call would lose its return value. So
	// args has to live in non-moving memory: we take it from the heap-backed
	// pool. The heap does not move on today's Go (non-compacting GC, confirmed
	// through Go 1.27). If a compacting GC ever lands, add runtime.Pinner here.
	args := syscallArgsPool.Get().(*syscallArgs)
	defer syscallArgsPool.Put(args)

	words := args.words
	if cap(words) < len(stack) {
		words = make([]uintptr, len(stack))
	}
	words = words[:len(stack)]
	copy(words, stack)

	*args = syscallArgs{
		fn: fn,
		a1: gpr[0], a2: gpr[1], a3: gpr[2],
		a4: gpr[3], a5: gpr[4], a6: gpr[5],
		nstack:  uintptr(len(words)),
		f1:      *(*uintptr)(unsafe.Pointer(&sse[0])),
		f2:      *(*uintptr)(unsafe.Pointer(&sse[1])),
		f3:      *(*uintptr)(unsafe.Pointer(&sse[2])),
		f4:      *(*uintptr)(unsafe.Pointer(&sse[3])),
		f5:      *(*uintptr)(unsafe.Pointer(&sse[4])),
		f6:      *(*uintptr)(unsafe.Pointer(&sse[5])),
		f7:      *(*uintptr)(unsafe.Pointer(&sse[6])),
		f8:      *(*uintptr)(unsafe.Pointer(&sse[7])),
		errnoFn: errnoFn,
		words:   words,
	}
	if len(words) > 0 {
		// The address of heap memory that args.words keeps alive and that the
		// GC does not move; syscallN copies the words before the C call.
		args.stack = uintptr(unsafe.Pointer(unsafe.SliceData(words)))
	}

	runtime_cgocall(syscallNABI0, unsafe.Pointer(args))

	r1 = args.r1
	r2 = args.r2
	f1 = *(*float64)(unsafe.Pointer(&args.f1))
	f2 = *(*float64)(unsafe.Pointer(&args.f2))
	cerrno = args.errno
	return
}
