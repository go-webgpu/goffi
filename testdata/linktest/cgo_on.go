// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build cgo && (linux || darwin || freebsd)

package main

import "linktest/cpkg"

func init() { cgoGetpid = cpkg.Getpid }
