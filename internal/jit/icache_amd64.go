// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build (linux && !android) || (darwin && !ios)

package jit

// flushICache does nothing on amd64, whose instruction fetch sees data
// writes without cache maintenance.
func flushICache([]byte) error { return nil }
