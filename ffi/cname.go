// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !windows && !goffi_static

package ffi

import (
	"strings"
	"syscall"
)

// checkCName rejects library and symbol names that contain a NUL byte.
// dlopen and dlsym take C strings, so such a name would be silently cut at
// the NUL and could load or resolve a different object. The error matches
// what syscall.BytePtrFromString (used on Windows) reports: syscall.EINVAL.
func checkCName(operation, name string) error {
	if strings.IndexByte(name, 0) >= 0 {
		return &LibraryError{Operation: operation, Name: name, Err: syscall.EINVAL}
	}
	return nil
}
