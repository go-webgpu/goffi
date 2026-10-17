// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build linux || darwin || freebsd

// Package goffiprobe calls into the C library through goffi.
package goffiprobe

import (
	"runtime"
	"syscall"
	"unsafe"

	"github.com/go-webgpu/goffi/ffi"
	"github.com/go-webgpu/goffi/types"
)

// LibC is the C library that provides getpid and close on this system.
func LibC() string {
	switch runtime.GOOS {
	case "darwin":
		return "/usr/lib/libSystem.B.dylib"
	case "freebsd":
		return "libc.so.7"
	default:
		return "libc.so.6"
	}
}

func symbol(name string) (unsafe.Pointer, error) {
	lib, err := ffi.LoadLibrary(LibC())
	if err != nil {
		return nil, err
	}
	return ffi.GetSymbol(lib, name)
}

// Getpid returns getpid() called through goffi.
func Getpid() (int32, error) {
	fn, err := symbol("getpid")
	if err != nil {
		return 0, err
	}
	var cif types.CallInterface
	if err := ffi.PrepareCallInterface(&cif, types.DefaultCall, types.SInt32TypeDescriptor, nil); err != nil {
		return 0, err
	}
	var pid int32
	if _, err := ffi.CallFunction(&cif, fn, unsafe.Pointer(&pid), nil); err != nil {
		return 0, err
	}
	return pid, nil
}

// CloseInvalidErrno calls close(-1) through goffi and returns its result and
// the errno goffi captured. The C library sets errno to EBADF.
func CloseInvalidErrno() (int32, syscall.Errno, error) {
	fn, err := symbol("close")
	if err != nil {
		return 0, 0, err
	}
	var cif types.CallInterface
	if err := ffi.PrepareCallInterface(&cif, types.DefaultCall, types.SInt32TypeDescriptor,
		[]*types.TypeDescriptor{types.SInt32TypeDescriptor}); err != nil {
		return 0, 0, err
	}
	fd := int32(-1)
	var ret int32
	errno, err := ffi.CallFunction(&cif, fn, unsafe.Pointer(&ret), []unsafe.Pointer{unsafe.Pointer(&fd)})
	return ret, errno, err
}
