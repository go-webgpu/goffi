//go:build darwin && amd64

package ffi

import (
	"errors"
	"unsafe"
)

var ErrUnsupportedPlatform = errors.New("macOS support not implemented")

func LoadLibrary(name string) (unsafe.Pointer, error) {
	return nil, ErrUnsupportedPlatform
}

func GetSymbol(handle unsafe.Pointer, name string) (unsafe.Pointer, error) {
	return nil, ErrUnsupportedPlatform
}
