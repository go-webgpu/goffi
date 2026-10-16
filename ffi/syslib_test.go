// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

package ffi

import (
	"runtime"
	"testing"

	"github.com/go-webgpu/goffi/types"
)

// systemConvention returns the calling convention of the C libraries below.
func systemConvention() types.CallingConvention {
	if runtime.GOOS == "windows" {
		return types.WindowsCallingConvention
	}
	return types.UnixCallingConvention
}

// System libraries the tests call into, by the name LoadLibrary takes on each
// platform. Keep the names here instead of a switch in every test, so a newly
// supported platform is added once and every test runs on it.

// systemLibC returns the C library: puts, strlen, abs, snprintf, printf.
func systemLibC(t *testing.T) string {
	t.Helper()
	switch runtime.GOOS {
	case "linux":
		return "libc.so.6"
	case "darwin":
		return "libSystem.B.dylib"
	case "freebsd":
		return "libc.so.7"
	case "windows":
		return "msvcrt.dll"
	}
	t.Skipf("no system C library name for %s", runtime.GOOS)
	return ""
}

// systemLibM returns the library with the C math functions (modff).
func systemLibM(t *testing.T) string {
	t.Helper()
	switch runtime.GOOS {
	case "linux":
		return "libm.so.6"
	case "darwin":
		return "libSystem.B.dylib"
	case "freebsd":
		return "libm.so.5"
	case "windows":
		return "msvcrt.dll"
	}
	t.Skipf("no system math library name for %s", runtime.GOOS)
	return ""
}
