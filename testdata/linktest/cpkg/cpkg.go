// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

// Package cpkg is an ordinary cgo package with no relation to goffi or
// purego. Linking it into a program makes cmd/link use the external linker.
package cpkg

// #include <unistd.h>
import "C"

// Getpid returns getpid() called through cgo.
func Getpid() int32 { return int32(C.getpid()) }
