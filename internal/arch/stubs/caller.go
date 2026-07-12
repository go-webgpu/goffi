//go:build !amd64 && !arm64

package stubs

import (
	"unsafe"

	"github.com/go-webgpu/goffi/internal/arch"
	"github.com/go-webgpu/goffi/types"
)

type unsupportedCaller struct{}

func init() {
	arch.Register(&unsupportedCaller{}, &unsupportedCaller{})
}

func (c *unsupportedCaller) Execute(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
	errnoFn uintptr,
) (cerrno uintptr, err error) {
	return 0, types.ErrUnsupportedArchitecture
}

func (c *unsupportedCaller) ClassifyReturn(
	t *types.TypeDescriptor,
	abi types.CallingConvention,
) int {
	return 0
}

func (c *unsupportedCaller) ClassifyArgument(
	t *types.TypeDescriptor,
	abi types.CallingConvention,
) arch.ArgumentClassification {
	return arch.ArgumentClassification{}
}
