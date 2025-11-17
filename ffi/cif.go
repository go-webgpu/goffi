package ffi

import (
	"fmt"
	"runtime"

	"github.com/go-webgpu/goffi/internal/arch"
	"github.com/go-webgpu/goffi/types"
)

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

	for _, arg := range cif.ArgTypes {
		classification := arch.Registry.Classifier.ClassifyArgument(arg, cif.Convention)
		gprCount += classification.GPRCount
		sseCount += classification.SSECount
		if gprCount > maxGPR || sseCount > maxSSE {
			// Handle register overflow
		}
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

// maxGPRegisters returns max general purpose registers
func maxGPRegisters(convention types.CallingConvention) int {
	if convention == types.UnixCallingConvention {
		return 6 // RDI, RSI, RDX, RCX, R8, R9
	}
	return 4 // Windows: RCX, RDX, R8, R9
}

// maxSSERegisters returns max SSE registers
func maxSSERegisters(convention types.CallingConvention) int {
	if convention == types.UnixCallingConvention {
		return 8 // XMM0-7
	}
	return 4 // Windows: XMM0-3
}
