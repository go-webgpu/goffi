//go:build arm64 && (linux || darwin || windows || freebsd)

// AAPCS64 ABI implementation (Linux, macOS, Windows, FreeBSD on ARM64)
// All ARM64 platforms use the same AAPCS64 calling convention for non-variadic
// functions; Apple platforms pack stack arguments differently (see layout.go).
// See: https://learn.microsoft.com/en-us/cpp/build/arm64-windows-abi-conventions

package arm64

import (
	"fmt"
	"math"
	"runtime"
	"unsafe"

	gosyscall "github.com/go-webgpu/goffi/internal/syscall"
	"github.com/go-webgpu/goffi/types"
)

// maxStackArgs is the number of 8-byte stack words the call trampoline copies.
const maxStackArgs = 7

// StackArgBytes reports how many bytes of stack the arguments of cif need on
// this OS and how many the call path supports. PrepareCallInterface uses it
// to reject calls that do not fit.
func (i *Implementation) StackArgBytes(cif *types.CallInterface) (need, limit uintptr, err error) {
	_, need, err = layoutArgs(cif, runtime.GOOS)
	return need, maxStackBytes, err
}

// Execute performs an AAPCS64 call: arguments are placed as layoutArgs says,
// the result is read back by handleReturn.
func (i *Implementation) Execute(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
	errnoFn uintptr,
) (cerrno uintptr, err error) {
	if len(avalue) < len(cif.ArgTypes) {
		return 0, fmt.Errorf("goffi: %d argument values for %d argument types", len(avalue), len(cif.ArgTypes))
	}
	locs, stackBytes, err := layoutArgs(cif, runtime.GOOS)
	if err != nil {
		return 0, err
	}
	if stackBytes > maxStackBytes {
		return 0, fmt.Errorf("goffi: arguments need %d bytes of stack, the arm64 call path supports %d",
			stackBytes, maxStackBytes)
	}

	var (
		gpr       [8]uintptr
		fpr       [8]uint64
		stackArgs [maxStackArgs]uintptr
		pinner    runtime.Pinner
	)
	defer pinner.Unpin()
	stackMem := unsafe.Slice((*byte)(unsafe.Pointer(&stackArgs[0])), maxStackBytes)
	gprMem := unsafe.Slice((*byte)(unsafe.Pointer(&gpr[0])), len(gpr)*8)

	for idx, loc := range locs {
		t := cif.ArgTypes[idx]
		val := avalue[idx]

		if loc.byRef {
			// A composite larger than 16 bytes is passed as a pointer to a
			// copy owned by the caller, so the callee may modify it without
			// touching the Go value. The copy is pinned for the call.
			cp := make([]byte, t.Size)
			copy(cp, unsafe.Slice((*byte)(val), t.Size))
			pinner.Pin(&cp[0])
			ptr := uintptr(unsafe.Pointer(&cp[0]))
			if loc.kind == locGPR {
				gpr[loc.reg] = ptr
			} else {
				*(*uintptr)(unsafe.Pointer(&stackMem[loc.offset])) = ptr
			}
			continue
		}

		switch loc.kind {
		case locGPR:
			if t.Kind == types.StructType {
				// Raw bytes in consecutive X registers.
				copy(gprMem[loc.reg*8:loc.reg*8+loc.nregs*8], unsafe.Slice((*byte)(val), t.Size))
			} else {
				gpr[loc.reg] = uintptr(loadInteger(t, val))
			}

		case locFPR:
			for k := 0; k < loc.nregs; k++ {
				elem := unsafe.Add(val, uintptr(k)*loc.elem)
				if loc.elem == 4 {
					fpr[loc.reg+k] = uint64(math.Float32bits(*(*float32)(elem)))
				} else {
					fpr[loc.reg+k] = math.Float64bits(*(*float64)(elem))
				}
			}

		case locStack:
			dst := stackMem[loc.offset:]
			if loc.widen {
				*(*uint64)(unsafe.Pointer(&dst[0])) = loadInteger(t, val)
			} else {
				copy(dst[:loc.size], unsafe.Slice((*byte)(val), loc.size))
			}
		}
	}

	// Determine if we need to pass X8 for large struct return (sret)
	var r8 uintptr
	if cif.Flags&types.ReturnViaPointer != 0 {
		// For sret, pass rvalue pointer in X8 - callee writes directly to it
		r8 = uintptr(rvalue)
	}

	// Call via our ARM64 syscall wrapper; errnoFn is non-zero on Unix, 0 on Windows.
	// When errnoFn is 0, the assembly skips errno capture (CBZ).
	stackWords := int(stackBytes / 8)
	ret1, ret2, fret, capturedErrno := gosyscall.CallNFloatErrno(uintptr(fn), gpr, fpr, stackArgs, stackWords, r8, errnoFn)

	runtime.KeepAlive(avalue)

	// Handle return value based on type
	return capturedErrno, i.handleReturn(cif, rvalue, uint64(ret1), uint64(ret2), fret)
}

// loadInteger reads an integer or pointer argument and sign- or zero-extends
// it to 64 bits.
func loadInteger(t *types.TypeDescriptor, p unsafe.Pointer) uint64 {
	switch t.Kind {
	case types.SInt8Type:
		return uint64(int64(*(*int8)(p)))
	case types.UInt8Type:
		return uint64(*(*uint8)(p))
	case types.SInt16Type:
		return uint64(int64(*(*int16)(p)))
	case types.UInt16Type:
		return uint64(*(*uint16)(p))
	case types.SInt32Type, types.IntType:
		return uint64(int64(*(*int32)(p)))
	case types.UInt32Type:
		return uint64(*(*uint32)(p))
	default: // SInt64Type, UInt64Type, PointerType
		return *(*uint64)(p)
	}
}
