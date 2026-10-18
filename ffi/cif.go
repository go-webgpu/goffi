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

	// Architectures that lay out the stack exactly (arm64: AAPCS64 and Apple
	// packing) decide whether the arguments fit.
	if layout, ok := arch.Registry.Caller.(arch.StackLayout); ok {
		need, limit, err := layout.StackArgBytes(cif)
		if err != nil {
			return err
		}
		if need > limit {
			return fmt.Errorf("%w: arguments need %d bytes of stack, platform supports %d",
				ErrTooManyArguments, need, limit)
		}
	} else if err := checkStackSlots(cif); err != nil {
		return err
	}

	// Windows-specific: requires 32-byte shadow space
	if cif.Convention == types.WindowsCallingConvention && cif.StackBytes < 32 {
		cif.StackBytes = 32
	}

	return nil
}

// callInterfaceForCall returns the CallInterface CallFunctionContext executes.
//
// PrepareCallInterface sets Flags for every struct result (no classification
// returns 0 for a struct, see TestPrepareCallInterface_StructReturnFlagsNonZero),
// so a struct result with Flags == 0 means the CallInterface was filled in by
// hand. Arguments are laid out from ArgTypes on every call either way, but a
// struct result is read according to Flags: the register pair of a 9-16 byte
// struct on System V amd64, HFA and indirect (X8) results on arm64. For such a
// CallInterface the result is classified here, on a copy: the caller's
// CallInterface and its type descriptors may be shared between goroutines and
// are never written. A CallInterface from PrepareCallInterface is returned as is.
func callInterfaceForCall(cif *types.CallInterface) (*types.CallInterface, error) {
	ret := cif.ReturnType
	if ret == nil {
		return nil, &InvalidCallInterfaceError{
			Field:  "returnType",
			Reason: "must not be nil",
			Index:  -1,
		}
	}
	if ret.Kind != types.StructType || cif.Flags != 0 {
		return cif, nil
	}
	if ret.Size == 0 || ret.Alignment == 0 {
		return nil, &InvalidCallInterfaceError{
			Field:  "returnType",
			Reason: "struct size and alignment are not set; prepare the call interface with PrepareCallInterface",
			Index:  -1,
		}
	}
	if arch.Registry.Classifier == nil {
		return nil, types.ErrUnsupportedArchitecture
	}
	prepared := *cif
	if prepared.Convention == types.DefaultCall {
		prepared.Convention = types.DefaultConvention()
	}
	prepared.Flags = arch.Registry.Classifier.ClassifyReturn(ret, prepared.Convention)
	return &prepared, nil
}

// checkStackSlots is the generic argument-count check: it counts the
// registers each argument needs and the 8-byte stack slots that overflow.
func checkStackSlots(cif *types.CallInterface) error {
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
