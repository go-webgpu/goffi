// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !cgo && linux && !android && !goffi_static

// Ported from runtime/cgo/linux.go and runtime/cgo/linux_syscall.c via
// github.com/ebitengine/purego (internal/fakecgo/linux.go, purego#426).
//
// Once fakecgo sets runtime.iscgo, syscall.Setuid and friends can no longer
// use the runtime's own all-threads syscall: runtime.doAllThreadsSyscall
// panics with "doAllThreadsSyscall not supported with cgo enabled". Like
// runtime/cgo, we provide the syscall.cgo_libc_set*id hooks so that the
// syscall package calls glibc instead, and glibc applies the change to every
// thread of the process.

package fakecgo

import (
	"structs"
	"unsafe"
)

// argset matches runtime/cgocall.go:argset, which in turn matches
// runtime/cgo/linux_syscall.c:argset_t.
type argset struct {
	_      structs.HostLayout
	args   *uintptr
	retval uintptr
}

//go:nosplit
//go:norace
func (a *argset) arg(i int) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Add(unsafe.Pointer(a.args), uintptr(i)*unsafe.Sizeof(uintptr(0))))
}

//go:linkname _cgo_libc_setegid syscall.cgo_libc_setegid
//go:linkname _cgo_libc_seteuid syscall.cgo_libc_seteuid
//go:linkname _cgo_libc_setgid syscall.cgo_libc_setgid
//go:linkname _cgo_libc_setregid syscall.cgo_libc_setregid
//go:linkname _cgo_libc_setresgid syscall.cgo_libc_setresgid
//go:linkname _cgo_libc_setresuid syscall.cgo_libc_setresuid
//go:linkname _cgo_libc_setreuid syscall.cgo_libc_setreuid
//go:linkname _cgo_libc_setuid syscall.cgo_libc_setuid
//go:linkname _cgo_libc_setgroups syscall.cgo_libc_setgroups

var _cgo_libc_setegid = &_cgo_goffi_setegid_trampoline
var _cgo_libc_seteuid = &_cgo_goffi_seteuid_trampoline
var _cgo_libc_setgid = &_cgo_goffi_setgid_trampoline
var _cgo_libc_setregid = &_cgo_goffi_setregid_trampoline
var _cgo_libc_setresgid = &_cgo_goffi_setresgid_trampoline
var _cgo_libc_setresuid = &_cgo_goffi_setresuid_trampoline
var _cgo_libc_setreuid = &_cgo_goffi_setreuid_trampoline
var _cgo_libc_setuid = &_cgo_goffi_setuid_trampoline
var _cgo_libc_setgroups = &_cgo_goffi_setgroups_trampoline

//go:nosplit
//go:norace
func errno() int32 {
	// this indirection is to avoid go vet complaining about possible misuse of unsafe.Pointer
	loc := __errno_location()
	return **(**int32)(unsafe.Pointer(&loc))
}

// setxidResult stores a libc set*id result the way linux_syscall.c does:
// 0 on success, errno on failure.
//
//go:nosplit
//go:norace
func setxidResult(c *argset, ret int32) {
	if ret == -1 {
		c.retval = uintptr(errno())
	} else {
		c.retval = uintptr(ret)
	}
}

//go:linkname _cgo_goffi_setegid_trampoline _cgo_goffi_setegid_trampoline
var _cgo_goffi_setegid_trampoline byte
var x_cgo_goffi_setegid_call = x_cgo_goffi_setegid

//go:nosplit
//go:norace
func x_cgo_goffi_setegid(c *argset) {
	setxidResult(c, setegid(uint32(uintptr(c.arg(0)))))
}

//go:linkname _cgo_goffi_seteuid_trampoline _cgo_goffi_seteuid_trampoline
var _cgo_goffi_seteuid_trampoline byte
var x_cgo_goffi_seteuid_call = x_cgo_goffi_seteuid

//go:nosplit
//go:norace
func x_cgo_goffi_seteuid(c *argset) {
	setxidResult(c, seteuid(uint32(uintptr(c.arg(0)))))
}

//go:linkname _cgo_goffi_setgid_trampoline _cgo_goffi_setgid_trampoline
var _cgo_goffi_setgid_trampoline byte
var x_cgo_goffi_setgid_call = x_cgo_goffi_setgid

//go:nosplit
//go:norace
func x_cgo_goffi_setgid(c *argset) {
	setxidResult(c, setgid(uint32(uintptr(c.arg(0)))))
}

//go:linkname _cgo_goffi_setregid_trampoline _cgo_goffi_setregid_trampoline
var _cgo_goffi_setregid_trampoline byte
var x_cgo_goffi_setregid_call = x_cgo_goffi_setregid

//go:nosplit
//go:norace
func x_cgo_goffi_setregid(c *argset) {
	setxidResult(c, setregid(uint32(uintptr(c.arg(0))), uint32(uintptr(c.arg(1)))))
}

//go:linkname _cgo_goffi_setresgid_trampoline _cgo_goffi_setresgid_trampoline
var _cgo_goffi_setresgid_trampoline byte
var x_cgo_goffi_setresgid_call = x_cgo_goffi_setresgid

//go:nosplit
//go:norace
func x_cgo_goffi_setresgid(c *argset) {
	setxidResult(c, setresgid(uint32(uintptr(c.arg(0))), uint32(uintptr(c.arg(1))), uint32(uintptr(c.arg(2)))))
}

//go:linkname _cgo_goffi_setresuid_trampoline _cgo_goffi_setresuid_trampoline
var _cgo_goffi_setresuid_trampoline byte
var x_cgo_goffi_setresuid_call = x_cgo_goffi_setresuid

//go:nosplit
//go:norace
func x_cgo_goffi_setresuid(c *argset) {
	setxidResult(c, setresuid(uint32(uintptr(c.arg(0))), uint32(uintptr(c.arg(1))), uint32(uintptr(c.arg(2)))))
}

//go:linkname _cgo_goffi_setreuid_trampoline _cgo_goffi_setreuid_trampoline
var _cgo_goffi_setreuid_trampoline byte
var x_cgo_goffi_setreuid_call = x_cgo_goffi_setreuid

//go:nosplit
//go:norace
func x_cgo_goffi_setreuid(c *argset) {
	setxidResult(c, setreuid(uint32(uintptr(c.arg(0))), uint32(uintptr(c.arg(1)))))
}

//go:linkname _cgo_goffi_setuid_trampoline _cgo_goffi_setuid_trampoline
var _cgo_goffi_setuid_trampoline byte
var x_cgo_goffi_setuid_call = x_cgo_goffi_setuid

//go:nosplit
//go:norace
func x_cgo_goffi_setuid(c *argset) {
	setxidResult(c, setuid(uint32(uintptr(c.arg(0)))))
}

//go:linkname _cgo_goffi_setgroups_trampoline _cgo_goffi_setgroups_trampoline
var _cgo_goffi_setgroups_trampoline byte
var x_cgo_goffi_setgroups_call = x_cgo_goffi_setgroups

//go:nosplit
//go:norace
func x_cgo_goffi_setgroups(c *argset) {
	setxidResult(c, setgroups(uint32(uintptr(c.arg(0))), (*uint32)(c.arg(1))))
}
