// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build cgo && (linux || darwin || freebsd)

// Command clib is built with -buildmode=c-shared and -buildmode=c-archive,
// which always use the external linker, to check that a library that uses
// goffi links in those modes.
package main

import "C"

import "linktest/goffiprobe"

// GoffiLinktestGetpid returns getpid() called through goffi, or -1.
//
//export GoffiLinktestGetpid
func GoffiLinktestGetpid() C.int {
	pid, err := goffiprobe.Getpid()
	if err != nil {
		return -1
	}
	return C.int(pid)
}

func main() {}
