// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !cgo && (darwin || freebsd || linux || netbsd) && !goffi_static

package fakecgo

import "unsafe" // for go:linkname, and to clear the hooks below

//go:linkname x_cgo_setenv_trampoline x_cgo_setenv_trampoline
//go:linkname _cgo_setenv runtime._cgo_setenv
var x_cgo_setenv_trampoline byte
var _cgo_setenv = &x_cgo_setenv_trampoline

//go:linkname x_cgo_unsetenv_trampoline x_cgo_unsetenv_trampoline
//go:linkname _cgo_unsetenv runtime._cgo_unsetenv
var x_cgo_unsetenv_trampoline byte
var _cgo_unsetenv = &x_cgo_unsetenv_trampoline

// dropLibcEnvHooks clears _cgo_setenv and _cgo_unsetenv for a process that has
// no libc (hostlibc.Missing).
//
// The runtime's setenv_c and unsetenv_c ask only whether these hooks are set,
// not whether the process is cgo, so clearing runtime.iscgo does not stop
// os.Setenv from calling x_cgo_setenv -- and that calls libc's setenv, which is
// unbound there: a jump to address 0 (f4 #1213, on a host whose preloaded
// library aborted the re-executed process, so the fallback ran and the first
// os.Setenv killed it). With the hooks nil the runtime updates only its own
// copy of the environment, which is all a process with no libc has.
//
// It writes through unsafe.Pointer because it runs from x_cgo_init, before the
// heap and the write barrier exist.
//
//go:nosplit
func dropLibcEnvHooks() {
	*(*uintptr)(unsafe.Pointer(&_cgo_setenv)) = 0
	*(*uintptr)(unsafe.Pointer(&_cgo_unsetenv)) = 0
}
