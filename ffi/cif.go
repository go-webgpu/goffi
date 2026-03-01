package ffi

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/go-webgpu/goffi/internal/arch"
	"github.com/go-webgpu/goffi/types"
)

// ErrTooManyArguments is returned when the argument count exceeds the platform
// limit of registers plus stack slots supported by the syscall layer.
var ErrTooManyArguments = errors.New("goffi: argument count exceeds platform limit")

// prepareCallInterfaceCore implements core call interface preparation
func prepareCallInterfaceCore(
	cif *types.CallInterface,
	convention types.CallingConvention,
	argCount int,
	returnType *types.TypeDescriptor,
	argTypes []*types.TypeDescriptor,
) error {
	// Auto-resolve DefaultCall to platform-specific convention
	if convention == types.DefaultCall {
		convention = types.DefaultConvention()
	}

	// Validate input parameters
	if convention < types.UnixCallingConvention || convention > types.GnuWindowsCallingConvention {
		return &CallingConventionError{
			Convention: int(convention),
			Platform:   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			Reason:     "value must be 1 (Unix), 2 (Windows), or 3 (GNU Windows)",
		}
	}

	cif.Convention = convention
	cif.ArgCount = argCount
	cif.ArgTypes = argTypes
	cif.ReturnType = returnType

	// Initialize composite types
	if returnType.Size == 0 && returnType.Kind == types.StructType {
		if err := initializeCompositeType(returnType); err != nil {
			return err
		}
	}

	if !isValidType(returnType) {
		return newInvalidTypeError("returnType", int(returnType.Kind), "unsupported type kind")
	}

	// Calculate stack size
	stackBytes := uintptr(0)
	for i, t := range argTypes {
		if t.Size == 0 && t.Kind == types.StructType {
			if err := initializeCompositeType(t); err != nil {
				return fmt.Errorf("argument type at index %d: %w", i, err)
			}
		}
		if !isValidType(t) {
			return newInvalidTypeAtIndexError("argTypes", int(t.Kind), i, "unsupported type kind")
		}
		stackBytes = align(stackBytes, t.Alignment)
		stackBytes += align(t.Size, 8)
	}
	cif.StackBytes = stackBytes

	return preparePlatformSpecific(cif)
}

// preparePlatformSpecific performs platform-specific preparation
func preparePlatformSpecific(cif *types.CallInterface) error {
	if arch.Registry.Classifier == nil {
		return types.ErrUnsupportedArchitecture
	}

	cif.Flags = arch.Registry.Classifier.ClassifyReturn(cif.ReturnType, cif.Convention)

	var gprCount, sseCount int
	maxGPR, maxSSE := maxGPRegisters(cif.Convention), maxSSERegisters(cif.Convention)
	maxStack := maxStackSlots(cif.Convention)

	for _, arg := range cif.ArgTypes {
		classification := arch.Registry.Classifier.ClassifyArgument(arg, cif.Convention)
		gprCount += classification.GPRCount
		sseCount += classification.SSECount
	}

	// Compute stack overflow counts
	gprStack := 0
	if gprCount > maxGPR {
		gprStack = gprCount - maxGPR
	}
	sseStack := 0
	if sseCount > maxSSE {
		sseStack = sseCount - maxSSE
	}
	totalStack := gprStack + sseStack

	if totalStack > maxStack {
		return fmt.Errorf("%w: %d args overflow to stack, platform supports %d stack slots",
			ErrTooManyArguments, totalStack, maxStack)
	}

	// Windows-specific: requires 32-byte shadow space
	if cif.Convention == types.WindowsCallingConvention && cif.StackBytes < 32 {
		cif.StackBytes = 32
	}

	return nil
}

// initializeCompositeType initializes composite type
func initializeCompositeType(t *types.TypeDescriptor) error {
	if t == nil {
		return &TypeValidationError{
			TypeName: "compositeType",
			Kind:     0,
			Reason:   "type descriptor is nil",
			Index:    -1,
		}
	}
	if t.Kind != types.StructType {
		return &TypeValidationError{
			TypeName: "compositeType",
			Kind:     int(t.Kind),
			Reason:   "expected StructType",
			Index:    -1,
		}
	}
	if t.Members == nil {
		return &TypeValidationError{
			TypeName: "compositeType",
			Kind:     int(t.Kind),
			Reason:   "struct has no members",
			Index:    -1,
		}
	}

	t.Size = 0
	t.Alignment = 0

	for i, member := range t.Members {
		if member.Size == 0 && member.Kind == types.StructType {
			if err := initializeCompositeType(member); err != nil {
				return fmt.Errorf("struct member at index %d: %w", i, err)
			}
		}
		if !isValidType(member) {
			return newInvalidTypeAtIndexError("structMember", int(member.Kind), i, "unsupported type kind")
		}

		t.Size = align(t.Size, member.Alignment)
		t.Size += member.Size

		if member.Alignment > t.Alignment {
			t.Alignment = member.Alignment
		}
	}

	t.Size = align(t.Size, t.Alignment)
	return nil
}

// isValidType validates type descriptor
func isValidType(t *types.TypeDescriptor) bool {
	switch t.Kind {
	case types.VoidType, types.IntType, types.FloatType, types.DoubleType,
		types.UInt8Type, types.SInt8Type, types.UInt16Type, types.SInt16Type,
		types.UInt32Type, types.SInt32Type, types.UInt64Type, types.SInt64Type,
		types.StructType, types.PointerType:
		return true
	default:
		return false
	}
}

// align aligns value to specified boundary
func align(value, alignment uintptr) uintptr {
	return (value + alignment - 1) &^ (alignment - 1)
}

// maxGPRegisters returns max general purpose registers for the calling convention.
func maxGPRegisters(convention types.CallingConvention) int {
	switch convention {
	case types.WindowsCallingConvention, types.GnuWindowsCallingConvention:
		return 4 // Windows Win64: RCX, RDX, R8, R9
	default:
		// System V AMD64: RDI, RSI, RDX, RCX, R8, R9 (6)
		// AAPCS64 (ARM64): X0-X7 (8)
		// At compile time the correct value is selected by arch-specific code;
		// here we use the AMD64 Unix default which is also used for overflow checking.
		return 6
	}
}

// maxSSERegisters returns max SSE/FP registers for the calling convention.
func maxSSERegisters(convention types.CallingConvention) int {
	switch convention {
	case types.WindowsCallingConvention, types.GnuWindowsCallingConvention:
		return 4 // Windows Win64: XMM0-XMM3
	default:
		return 8 // System V AMD64 / AAPCS64: XMM0-7 / D0-D7
	}
}

// maxStackSlots returns the maximum number of additional stack argument slots
// supported by the platform-specific syscall layer.
func maxStackSlots(convention types.CallingConvention) int {
	// Our syscall layer supports up to maxArgs (15) total args:
	// Unix AMD64:  6 GP registers + 9 stack slots = 15
	// Unix ARM64:  8 GP registers + 7 stack slots = 15
	// Windows:     syscall.SyscallN supports up to 15 args natively
	return 9
}
