package ffi

import (
	"unsafe"

	"github.com/go-webgpu/goffi/internal/arch"
	"github.com/go-webgpu/goffi/types"
)

// executeFunction вызывает функцию через архитектурно-зависимый механизм
func executeFunction(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
) error {
	if arch.Registry.Caller == nil {
		return types.ErrUnsupportedArchitecture
	}
	return arch.Registry.Caller.Execute(cif, fn, rvalue, avalue)
}
