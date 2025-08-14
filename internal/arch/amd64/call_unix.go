//go:build amd64 && (linux || darwin)

package amd64

import (
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

func (i *Implementation) Execute(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
) error {
	// Allocate stack space (aligned to 16 bytes)
	stackSize := cif.StackBytes
	if stackSize%16 != 0 {
		stackSize = (stackSize/16 + 1) * 16
	}
	stack := make([]byte, stackSize)
	stackPtr := unsafe.Pointer(&stack[0])

	// Prepare registers
	gprRegs := make([]uint64, 6)  // RDI, RSI, RDX, RCX, R8, R9
	sseRegs := make([]float64, 8) // XMM0-7

	gprIndex, sseIndex, stackOffset := 0, 0, uintptr(0)

	// Marshal arguments
	for idx, argType := range cif.ArgTypes {
		classification := i.ClassifyArgument(argType, cif.Convention)

		// Try to pass in registers
		if classification.GPRCount > 0 && gprIndex+classification.GPRCount <= len(gprRegs) {
			for j := 0; j < classification.GPRCount; j++ {
				gprRegs[gprIndex] = *(*uint64)(avalue[idx])
				gprIndex++
			}
			continue
		}

		if classification.SSECount > 0 && sseIndex+classification.SSECount <= len(sseRegs) {
			for j := 0; j < classification.SSECount; j++ {
				sseRegs[sseIndex] = *(*float64)(avalue[idx])
				sseIndex++
			}
			continue
		}

		// Pass on stack
		size := i.align(argType.Size, 8)
		copy(stack[stackOffset:stackOffset+size], (*(*[1 << 30]byte)(avalue[idx]))[:size])
		stackOffset += size
	}

	// Call via assembly
	retVal := callUnix64(gprRegs, sseRegs, fn, stackPtr)

	// Handle return value
	return i.handleReturn(cif, rvalue, retVal)
}

func callUnix64(gpr []uint64, sse []float64, fn, stack unsafe.Pointer) uint64
