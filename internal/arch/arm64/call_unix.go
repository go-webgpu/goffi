//go:build arm64 && (linux || darwin)

// Unix implementation using AAPCS64 ABI (Linux, macOS on ARM64)
// This implementation follows the ARM64 Procedure Call Standard.

package arm64

import (
	"math"
	"unsafe"

	gosyscall "github.com/go-webgpu/goffi/internal/syscall"
	"github.com/go-webgpu/goffi/types"
)

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
	// Prepare register arguments following AAPCS64
	// X0-X7: 8 integer/pointer registers
	// D0-D7: 8 floating-point registers
	var gpr [8]uintptr
	var fpr [8]uint64

	gprIdx := 0
	fprIdx := 0

	// Map arguments to registers
	for idx, argType := range cif.ArgTypes {
		if idx >= len(avalue) {
			break
		}

		switch argType.Kind {
		case types.FloatType:
			if fprIdx < 8 {
				bits := math.Float32bits(*(*float32)(avalue[idx]))
				fpr[fprIdx] = uint64(bits) // stored in low 32 bits
				fprIdx++
			}
		case types.DoubleType:
			if fprIdx < 8 {
				bits := math.Float64bits(*(*float64)(avalue[idx]))
				fpr[fprIdx] = bits
				fprIdx++
			}
		case types.PointerType:
			if gprIdx < 8 {
				gpr[gprIdx] = *(*uintptr)(avalue[idx])
				gprIdx++
			}
		case types.SInt8Type:
			if gprIdx < 8 {
				gpr[gprIdx] = uintptr(int64(*(*int8)(avalue[idx])))
				gprIdx++
			}
		case types.UInt8Type:
			if gprIdx < 8 {
				gpr[gprIdx] = uintptr(*(*uint8)(avalue[idx]))
				gprIdx++
			}
		case types.SInt16Type:
			if gprIdx < 8 {
				gpr[gprIdx] = uintptr(int64(*(*int16)(avalue[idx])))
				gprIdx++
			}
		case types.UInt16Type:
			if gprIdx < 8 {
				gpr[gprIdx] = uintptr(*(*uint16)(avalue[idx]))
				gprIdx++
			}
		case types.SInt32Type:
			if gprIdx < 8 {
				gpr[gprIdx] = uintptr(int64(*(*int32)(avalue[idx])))
				gprIdx++
			}
		case types.UInt32Type:
			if gprIdx < 8 {
				gpr[gprIdx] = uintptr(*(*uint32)(avalue[idx]))
				gprIdx++
			}
		case types.SInt64Type:
			if gprIdx < 8 {
				gpr[gprIdx] = uintptr(*(*int64)(avalue[idx]))
				gprIdx++
			}
		case types.UInt64Type:
			if gprIdx < 8 {
				gpr[gprIdx] = uintptr(*(*uint64)(avalue[idx]))
				gprIdx++
			}
		case types.StructType:
			// AAPCS64:
			// - HFA (1-4 floats/doubles): passed in D registers
			// - <=16 bytes non-HFA: passed in X registers (1 or 2 registers)
			// - >16 bytes: passed by reference
			ensureStructLayout(argType)

			isHFA, hfaCount, _ := isHomogeneousFloatAggregate(argType)
			if isHFA && hfaCount > 0 && hfaCount <= 4 && fprIdx+hfaCount <= 8 {
				ok := placeStructRegisters(
					avalue[idx],
					argType,
					func(v uint64) bool {
						if gprIdx >= 8 {
							return false
						}
						gpr[gprIdx] = uintptr(v)
						gprIdx++
						return true
					},
					func(v uint64) bool {
						if fprIdx >= 8 {
							return false
						}
						fpr[fprIdx] = v
						fprIdx++
						return true
					},
				)
				if ok {
					break
				}
			}

			if argType.Size <= 16 {
				intCount, floatCount := countStructRegUsage(argType)
				if gprIdx+intCount <= 8 && fprIdx+floatCount <= 8 {
					ok := placeStructRegisters(
						avalue[idx],
						argType,
						func(v uint64) bool {
							if gprIdx >= 8 {
								return false
							}
							gpr[gprIdx] = uintptr(v)
							gprIdx++
							return true
						},
						func(v uint64) bool {
							if fprIdx >= 8 {
								return false
							}
							fpr[fprIdx] = v
							fprIdx++
							return true
						},
					)
					if ok {
						break
					}
				}
			}

			// Fallback: pass by reference.
			if gprIdx < 8 {
				gpr[gprIdx] = uintptr(avalue[idx])
				gprIdx++
			}
		default:
			// For unknown types, pass as pointer
			if gprIdx < 8 {
				gpr[gprIdx] = uintptr(avalue[idx])
				gprIdx++
			}
		}
	}

	// Determine if we need to pass r8 for large struct return (sret)
	var r8 uintptr
	if cif.Flags&types.ReturnViaPointer != 0 && rvalue != nil {
		// For sret, pass rvalue pointer in X8 - callee writes directly to it
		r8 = uintptr(rvalue)
	}

	// Call via our ARM64 syscall wrapper
	ret1, ret2, fret := gosyscall.Call8Float(uintptr(fn), gpr, fpr, r8)

	// Handle return value based on type
	return i.handleReturn(cif, rvalue, uint64(ret1), uint64(ret2), fret)
}
