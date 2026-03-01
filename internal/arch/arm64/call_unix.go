//go:build arm64 && (linux || darwin)

// Unix implementation using AAPCS64 ABI (Linux, macOS on ARM64)
// This implementation follows the ARM64 Procedure Call Standard.

package arm64

import (
	"fmt"
	"math"
	"runtime"
	"unsafe"

	gosyscall "github.com/go-webgpu/goffi/internal/syscall"
	"github.com/go-webgpu/goffi/types"
)

// maxStackArgs is the number of stack spill slots supported in the syscall ABI.
const maxStackArgs = 7

func placeStructRegisters(
	base unsafe.Pointer,
	desc *types.TypeDescriptor,
	addInt func(uint64) bool,
	addFloat func(uint64) bool,
) bool {
	if base == nil || desc == nil || desc.Kind != types.StructType {
		return false
	}

	var (
		val   uint64
		shift uint
		class regClass
		ok    = true
	)

	flush := func() {
		if !ok || class == classNone {
			val = 0
			shift = 0
			class = classNone
			return
		}
		if class == classFloat {
			ok = addFloat(val) && ok
		} else {
			ok = addInt(val) && ok
		}
		val = 0
		shift = 0
		class = classNone
	}

	var place func(cur *types.TypeDescriptor, ptr unsafe.Pointer)
	place = func(cur *types.TypeDescriptor, ptr unsafe.Pointer) {
		if !ok || cur == nil {
			return
		}
		if cur.Kind == types.StructType {
			offset := uintptr(0)
			for _, member := range cur.Members {
				if member == nil {
					continue
				}
				offset = alignOffset(offset, member.Alignment)
				place(member, unsafe.Add(ptr, offset))
				offset += member.Size
			}
			return
		}

		alignBits := uint(cur.Alignment*8 - 1)
		shift = (shift + alignBits) &^ alignBits
		if shift >= 64 {
			flush()
			shift = 0
		}

		switch cur.Kind {
		case types.FloatType:
			if class == classFloat {
				flush()
			}
			bits := math.Float32bits(*(*float32)(ptr))
			val |= uint64(bits) << shift
			shift += 32
			class |= classFloat
		case types.DoubleType:
			ok = addFloat(math.Float64bits(*(*float64)(ptr))) && ok
			shift = 0
			class = classNone
			val = 0
		case types.UInt8Type:
			val |= uint64(*(*uint8)(ptr)) << shift
			shift += 8
			class |= classInt
		case types.SInt8Type:
			val |= uint64(uint8(*(*int8)(ptr))) << shift
			shift += 8
			class |= classInt
		case types.UInt16Type:
			val |= uint64(*(*uint16)(ptr)) << shift
			shift += 16
			class |= classInt
		case types.SInt16Type:
			val |= uint64(uint16(*(*int16)(ptr))) << shift
			shift += 16
			class |= classInt
		case types.UInt32Type:
			val |= uint64(*(*uint32)(ptr)) << shift
			shift += 32
			class |= classInt
		case types.SInt32Type:
			val |= uint64(uint32(*(*int32)(ptr))) << shift
			shift += 32
			class |= classInt
		case types.UInt64Type:
			ok = addInt(*(*uint64)(ptr)) && ok
			shift = 0
			class = classNone
			val = 0
		case types.SInt64Type:
			ok = addInt(uint64(*(*int64)(ptr))) && ok
			shift = 0
			class = classNone
			val = 0
		case types.PointerType:
			ok = addInt(uint64(*(*uintptr)(ptr))) && ok
			shift = 0
			class = classNone
			val = 0
		default:
			ok = false
		}

		if !ok {
			return
		}
	}

	place(desc, base)
	if ok && class != classNone {
		flush()
	}
	return ok
}

func (i *Implementation) Execute(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
) error {
	// AAPCS64 ABI:
	// - X0-X7: 8 integer/pointer GP registers
	// - D0-D7: 8 floating-point registers
	// - Stack:  args 9+ (GP) or FP overflow

	var gpr [8]uintptr
	var fpr [8]uint64
	var stackArgs [maxStackArgs]uintptr

	gprIdx := 0
	fprIdx := 0
	stackIdx := 0

	addInt := func(x uintptr) bool {
		if gprIdx < 8 {
			gpr[gprIdx] = x
			gprIdx++
			return true
		}
		if stackIdx < maxStackArgs {
			stackArgs[stackIdx] = x
			stackIdx++
			return true
		}
		return false
	}

	addFloat := func(x uint64) bool {
		if fprIdx < 8 {
			fpr[fprIdx] = x
			fprIdx++
			return true
		}
		// AAPCS64: float overflow goes to stack as a full 8-byte slot
		if stackIdx < maxStackArgs {
			stackArgs[stackIdx] = uintptr(x)
			stackIdx++
			return true
		}
		return false
	}

	// Determine if we need to pass X8 for large struct return (sret)
	var r8 uintptr
	if cif.Flags&types.ReturnViaPointer != 0 && rvalue != nil {
		// For sret, pass rvalue pointer in X8 - callee writes directly to it
		r8 = uintptr(rvalue)
	}

	// Map arguments to registers or stack
	for idx, argType := range cif.ArgTypes {
		if idx >= len(avalue) {
			break
		}

		switch argType.Kind {
		case types.FloatType:
			// Use math.Float32bits to preserve exact 32-bit IEEE-754 pattern.
			addFloat(uint64(math.Float32bits(*(*float32)(avalue[idx]))))
		case types.DoubleType:
			addFloat(math.Float64bits(*(*float64)(avalue[idx])))
		case types.PointerType:
			addInt(*(*uintptr)(avalue[idx]))
		case types.SInt8Type:
			addInt(uintptr(int64(*(*int8)(avalue[idx]))))
		case types.UInt8Type:
			addInt(uintptr(*(*uint8)(avalue[idx])))
		case types.SInt16Type:
			addInt(uintptr(int64(*(*int16)(avalue[idx]))))
		case types.UInt16Type:
			addInt(uintptr(*(*uint16)(avalue[idx])))
		case types.SInt32Type:
			addInt(uintptr(int64(*(*int32)(avalue[idx]))))
		case types.UInt32Type:
			addInt(uintptr(*(*uint32)(avalue[idx])))
		case types.SInt64Type:
			addInt(uintptr(*(*int64)(avalue[idx])))
		case types.UInt64Type:
			addInt(uintptr(*(*uint64)(avalue[idx])))
		case types.StructType:
			// AAPCS64:
			// - HFA (1-4 floats/doubles): passed in D registers; if no room → entire HFA on stack
			// - <=16 bytes non-HFA: passed in X registers (1 or 2)
			// - >16 bytes non-HFA: passed by reference
			ensureStructLayout(argType)

			isHFA, hfaCount, _ := isHomogeneousFloatAggregate(argType)
			if isHFA && hfaCount > 0 && hfaCount <= 4 {
				if fprIdx+hfaCount <= 8 {
					// Fits in FP registers
					ok := placeStructRegisters(
						avalue[idx],
						argType,
						func(v uint64) bool { return addInt(uintptr(v)) },
						func(v uint64) bool { return addFloat(v) },
					)
					if ok {
						break
					}
				}
				// AAPCS64 HFA overflow rule: if HFA does not fit in remaining FP registers,
				// the entire HFA goes onto the stack (not split between regs and stack).
				// Each element occupies one 8-byte stack slot.
				hfaOverflow := false
				for k := 0; k < int(argType.Size/4) && !hfaOverflow; k++ {
					slot := *(*uint32)(unsafe.Add(avalue[idx], uintptr(k)*4))
					if !addFloat(uint64(slot)) {
						hfaOverflow = true
					}
				}
				if !hfaOverflow {
					break
				}
				// Fallthrough: pass entire struct on stack by reference as last resort
				addInt(uintptr(avalue[idx]))
				break
			}

			if argType.Size <= 16 {
				intCount, floatCount := countStructRegUsage(argType)
				if gprIdx+intCount <= 8 && fprIdx+floatCount <= 8 {
					ok := placeStructRegisters(
						avalue[idx],
						argType,
						func(v uint64) bool { return addInt(uintptr(v)) },
						func(v uint64) bool { return addFloat(v) },
					)
					if ok {
						break
					}
				}
			}

			// Fallback: pass by reference (pointer to value)
			addInt(uintptr(avalue[idx]))
		default:
			// For unknown types, pass as pointer
			addInt(uintptr(avalue[idx]))
		}
	}

	// Validate we haven't exceeded platform maximum
	if stackIdx > maxStackArgs {
		return fmt.Errorf("goffi: %d stack arguments exceed platform limit of %d", stackIdx, maxStackArgs)
	}

	// Call via our ARM64 syscall wrapper
	ret1, ret2, fret := gosyscall.CallNFloat(uintptr(fn), gpr, fpr, stackArgs, stackIdx, r8)

	runtime.KeepAlive(avalue)

	// Handle return value based on type
	return i.handleReturn(cif, rvalue, uint64(ret1), uint64(ret2), fret)
}
