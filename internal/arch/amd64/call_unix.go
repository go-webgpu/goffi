//go:build amd64 && (linux || darwin || freebsd)

// Unix implementation using System V AMD64 ABI (Linux, macOS, FreeBSD, etc.)
// This implementation closely follows purego's proven approach but is OUR OWN code.

package amd64

import (
	"fmt"
	"math"
	"runtime"
	"unsafe"

	gosyscall "github.com/go-webgpu/goffi/internal/syscall"
	"github.com/go-webgpu/goffi/types"
)

// maxTotalArgs is the maximum number of GP + stack argument slots supported.
// Matches purego's maxArgs = 15.
const maxTotalArgs = 15

func (i *Implementation) Execute(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
) error {
	// System V AMD64 ABI:
	// - GP registers: RDI, RSI, RDX, RCX, R8, R9 (6 registers, indices 0-5)
	// - SSE registers: XMM0-XMM7 (8 registers)
	// - Stack args: additional GP/integer args beyond register count
	//
	// In our syscallArgs layout:
	//   sysargs[0..5]  -> RDI..R9 (6 GP registers)
	//   sysargs[6..14] -> stack slots (pushed before CALL)
	var sysargs [maxTotalArgs]uintptr
	var floats [8]uintptr

	numInts := 0   // GP register index (0-5 = registers, 6+ = stack)
	numFloats := 0 // SSE register index (0-7)
	numStack := 0  // stack slot index

	addInt := func(x uintptr) {
		const maxGPRegs = 6
		if numInts < maxGPRegs {
			sysargs[numInts] = x
			numInts++
		} else {
			// Overflow to stack: placed after the 6 register slots
			sysargs[maxGPRegs+numStack] = x
			numStack++
		}
	}

	addFloat := func(x uintptr) {
		if numFloats < 8 {
			floats[numFloats] = x
			numFloats++
		} else {
			// Float overflow to stack (each float occupies one 8-byte stack slot)
			const maxGPRegs = 6
			sysargs[maxGPRegs+numStack] = x
			numStack++
		}
	}

	// Detect sret: struct > 16 bytes requires hidden first argument in RDI.
	// The caller's rvalue buffer is passed as the first integer argument and
	// callee writes the return value directly into it.
	sretBuf := unsafe.Pointer(nil)
	if cif.ReturnType.Kind == types.StructType && cif.ReturnType.Size > 16 {
		if rvalue != nil {
			sretBuf = rvalue
		} else {
			sretBuf = unsafe.Pointer(&[128]byte{})
		}
		addInt(uintptr(sretBuf))
	}

	// Map arguments to registers or stack
	for idx, argType := range cif.ArgTypes {
		if idx >= len(avalue) {
			break
		}

		switch argType.Kind {
		case types.FloatType:
			// Use math.Float32bits to preserve exact bit pattern in XMM register.
			// Widening float32 → float64 corrupts the lower 32 bits read by callee.
			addFloat(uintptr(math.Float32bits(*(*float32)(avalue[idx]))))
		case types.DoubleType:
			addFloat(*(*uintptr)(avalue[idx]))
		case types.PointerType:
			addInt(*(*uintptr)(avalue[idx]))
		case types.SInt8Type, types.UInt8Type:
			addInt(uintptr(*(*uint8)(avalue[idx])))
		case types.SInt16Type, types.UInt16Type:
			addInt(uintptr(*(*uint16)(avalue[idx])))
		case types.SInt32Type, types.UInt32Type:
			addInt(uintptr(*(*uint32)(avalue[idx])))
		case types.SInt64Type, types.UInt64Type:
			addInt(uintptr(*(*uint64)(avalue[idx])))
		case types.StructType:
			argPtr := avalue[idx]
			sz := argType.Size
			switch {
			case sz == 0:
				// Zero-size struct: pass nothing.
			case sz <= 8:
				// Single eightbyte: INTEGER if any member is not float/double, else SSE.
				if isStructAllFloats(argType) {
					addFloat(*(*uintptr)(argPtr))
				} else {
					// Read only the bytes present to avoid overread.
					var v uintptr
					switch {
					case sz == 1:
						v = uintptr(*(*uint8)(argPtr))
					case sz == 2:
						v = uintptr(*(*uint16)(argPtr))
					case sz <= 4:
						v = uintptr(*(*uint32)(argPtr))
					default:
						v = *(*uintptr)(argPtr)
					}
					addInt(v)
				}
			case sz <= 16:
				// Two eightbytes: classify each independently.
				// System V ABI §3.2.3: INTEGER wins over SSE within an eightbyte.
				if classifyEightbyte(argType, 0, 8) {
					addFloat(*(*uintptr)(argPtr))
				} else {
					addInt(*(*uintptr)(argPtr))
				}
				remaining := sz - 8
				secondPtr := unsafe.Add(argPtr, 8)
				if classifyEightbyte(argType, 8, sz) {
					var v uintptr
					switch {
					case remaining == 1:
						v = uintptr(*(*uint8)(secondPtr))
					case remaining == 2:
						v = uintptr(*(*uint16)(secondPtr))
					case remaining <= 4:
						v = uintptr(*(*uint32)(secondPtr))
					default:
						v = *(*uintptr)(secondPtr)
					}
					addFloat(v)
				} else {
					var v uintptr
					switch {
					case remaining == 1:
						v = uintptr(*(*uint8)(secondPtr))
					case remaining == 2:
						v = uintptr(*(*uint16)(secondPtr))
					case remaining <= 4:
						v = uintptr(*(*uint32)(secondPtr))
					default:
						v = *(*uintptr)(secondPtr)
					}
					addInt(v)
				}
			default:
				// MEMORY class (> 16 bytes): copy onto stack in 8-byte chunks.
				nChunks := (sz + 7) / 8
				for k := uintptr(0); k < nChunks; k++ {
					chunkPtr := unsafe.Add(argPtr, k*8)
					bytesLeft := sz - k*8
					var v uintptr
					if bytesLeft >= 8 {
						v = *(*uintptr)(chunkPtr)
					} else {
						switch {
						case bytesLeft == 1:
							v = uintptr(*(*uint8)(chunkPtr))
						case bytesLeft == 2:
							v = uintptr(*(*uint16)(chunkPtr))
						case bytesLeft <= 4:
							v = uintptr(*(*uint32)(chunkPtr))
						default:
							v = *(*uintptr)(chunkPtr)
						}
					}
					addInt(v)
				}
			}
		default:
			// For unknown/composite types, pass as pointer-to-value
			addInt(uintptr(avalue[idx]))
		}
	}

	// Validate we haven't exceeded platform maximum
	if numStack > maxTotalArgs-6 {
		return fmt.Errorf("goffi: %d stack arguments exceed platform limit of %d", numStack, maxTotalArgs-6)
	}

	// Build GP register array (first 6 slots)
	var gpr [6]uintptr
	copy(gpr[:], sysargs[:6])

	// Build SSE array as float64 bit-patterns
	var sse [8]float64
	for k := range floats {
		sse[k] = *(*float64)(unsafe.Pointer(&floats[k]))
	}

	// Build stack slots (sysargs[6..14])
	var stackArgs [9]uintptr
	copy(stackArgs[:], sysargs[6:])

	// Call via syscall
	ret, r2, fret := gosyscall.CallNFloat(uintptr(fn), gpr, sse, stackArgs, numStack)

	runtime.KeepAlive(avalue)
	runtime.KeepAlive(sretBuf)

	// If sret, the callee wrote directly into rvalue — no further copy needed.
	if sretBuf != nil {
		return nil
	}

	// Handle return value based on type
	retVal := uint64(ret)

	// For float returns, use the float value from XMM0
	if cif.ReturnType.Kind == types.FloatType || cif.ReturnType.Kind == types.DoubleType {
		retVal = *(*uint64)(unsafe.Pointer(&fret))
	}

	return i.handleReturn(cif, rvalue, retVal, uint64(r2))
}

// isStructAllFloats returns true if every member of a flat struct is float or double.
// Per System V AMD64 ABI §3.2.3: if any member in an eightbyte is INTEGER class,
// the entire eightbyte is classified as INTEGER (INTEGER wins over SSE).
func isStructAllFloats(t *types.TypeDescriptor) bool {
	if len(t.Members) == 0 {
		return false
	}
	for _, m := range t.Members {
		if m.Kind != types.FloatType && m.Kind != types.DoubleType {
			return false
		}
	}
	return true
}

// classifyEightbyte returns true if all struct fields whose offset falls within
// [startOff, endOff) are SSE types (float or double).
// Returns false if any field in the range is INTEGER class, or if no fields lie in the range.
func classifyEightbyte(t *types.TypeDescriptor, startOff, endOff uintptr) bool {
	var offset uintptr
	allFloat := true
	hasField := false
	for _, m := range t.Members {
		if m == nil {
			continue
		}
		// Align offset to member alignment requirement.
		if m.Alignment > 0 {
			offset = (offset + m.Alignment - 1) &^ (m.Alignment - 1)
		}
		if offset >= startOff && offset < endOff {
			hasField = true
			if m.Kind != types.FloatType && m.Kind != types.DoubleType {
				allFloat = false
				break
			}
		}
		offset += m.Size
	}
	return hasField && allFloat
}
