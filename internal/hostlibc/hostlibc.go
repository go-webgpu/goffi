// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

// Package hostlibc records whether the running process actually has a libc.
//
// It exists for the universal ("Profile U") build, which is the only mode in
// which a goffi binary can start on a system where it cannot reach one. Such a
// binary names no libc and has no ELF interpreter, so the kernel loads it
// directly and goffi brings libc up itself by re-executing through the host's
// dynamic loader. On a host with no loader goffi recognises -- a scratch
// container, a distribution that keeps ld.so somewhere else -- there is nothing
// to re-execute through, and the process runs on with every imported libc
// symbol unbound.
//
// Missing says so. Setting it lets startup drop back to a pure-Go runtime
// instead of calling into unbound symbols, and lets the FFI entry points fail
// with an error rather than a fault.
//
// This package deliberately imports nothing. It is written from x_cgo_init,
// before the Go runtime is up: only a plain store to a package-level variable
// is safe there.
package hostlibc

// Missing reports that this process has no libc, so nothing that needs one --
// dlopen, dlsym, any foreign call -- can work.
//
// It is set once, from the universal build's re-exec bridge, before the Go
// runtime starts and therefore before anything can read it; it is false in
// every other build and on every host where the bridge succeeds. Nothing
// clears it, so no synchronisation is needed to read it.
var Missing bool

// missingError is a distinct type so ErrMissing needs no initialization at
// run time: the compiler lays the value out statically, which keeps this
// package free of an init function on the startup path.
type missingError struct{}

func (missingError) Error() string {
	return "goffi: universal build: this system has no dynamic loader goffi recognises, " +
		"so the process could not bind a libc and FFI is unavailable"
}

// ErrMissing is returned, usually wrapped in a *ffi.LibraryError, by every
// operation that needs libc when Missing is set: library loading, symbol
// lookup and foreign calls.
var ErrMissing error = missingError{}
