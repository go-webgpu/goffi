package ffi

import (
	"testing"

	"github.com/go-webgpu/goffi/types"
)

func BenchmarkPrepCIF(b *testing.B) {
	cif := &types.CallInterface{}
	rtype := types.VoidTypeDescriptor
	argtypes := []*types.TypeDescriptor{types.PointerTypeDescriptor}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PrepareCallInterface(cif, types.UnixCallingConvention, 1, rtype, argtypes)
	}
}
