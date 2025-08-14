package ffi

import (
	"errors"
	_ "github.com/go-webgpu/goffi/internal/arch/amd64" // Register amd64 implementation
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// FFI errors
var (
	ErrInvalidCallInterface = errors.New("invalid call interface")
	ErrFunctionCallFailed   = errors.New("function call failed")
)

// PrepareCallInterface prepares a function call interface
func PrepareCallInterface(
	cif *types.CallInterface,
	convention types.CallingConvention,
	argCount int,
	returnType *types.TypeDescriptor,
	argTypes []*types.TypeDescriptor,
) error {
	if cif == nil || returnType == nil || (argCount > 0 && argTypes == nil) {
		return ErrInvalidCallInterface
	}
	return prepareCallInterfaceCore(cif, convention, argCount, returnType, argTypes)
}

// CallFunction executes a function call
func CallFunction(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
) error {
	if cif == nil || fn == nil {
		return ErrInvalidCallInterface
	}
	return executeFunction(cif, fn, rvalue, avalue)
}
