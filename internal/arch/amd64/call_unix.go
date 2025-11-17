//go:build amd64 && (linux || darwin)

// Unix implementation using System V AMD64 ABI (Linux, macOS, FreeBSD, etc.)
// This implementation closely follows purego's proven approach but is OUR OWN code.

package amd64

import (
	"unsafe"

	"github.com/go-webgpu/goffi/internal/syscall"
	"github.com/go-webgpu/goffi/types"
)

func (i *Implementation) Execute(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
) error {
	// Prepare register arguments following System V AMD64 ABI
	var gpr [6]uintptr
	var sse [8]float64

	gprIdx := 0
	sseIdx := 0

	// Map arguments to registers
	for idx, argType := range cif.ArgTypes {
		if idx >= len(avalue) {
			break
		}

		switch argType.Kind {
		case types.FloatType:
			if sseIdx < 8 {
				sse[sseIdx] = float64(*(*float32)(avalue[idx]))
				sseIdx++
			}
		case types.DoubleType:
			if sseIdx < 8 {
				sse[sseIdx] = *(*float64)(avalue[idx])
				sseIdx++
			}
		case types.PointerType:
			if gprIdx < 6 {
				gpr[gprIdx] = uintptr(avalue[idx])
				gprIdx++
			}
		case types.SInt32Type, types.UInt32Type:
			if gprIdx < 6 {
				gpr[gprIdx] = uintptr(*(*uint32)(avalue[idx]))
				gprIdx++
			}
		case types.SInt64Type, types.UInt64Type:
			if gprIdx < 6 {
				gpr[gprIdx] = uintptr(*(*uint64)(avalue[idx]))
				gprIdx++
			}
		default:
			// For unknown types, pass as pointer
			if gprIdx < 6 {
				gpr[gprIdx] = uintptr(avalue[idx])
				gprIdx++
			}
		}
	}

	// Call via our syscall6
	ret, fret := syscall.Call6Float(uintptr(fn), gpr, sse)

	// Handle return value based on type
	retVal := uint64(ret)

	// For float returns, use the float value
	if cif.ReturnType.Kind == types.FloatType || cif.ReturnType.Kind == types.DoubleType {
		retVal = *(*uint64)(unsafe.Pointer(&fret))
	}

	return i.handleReturn(cif, rvalue, retVal)
}
