// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build android && arm64

package ffi

// NewCallback is intentionally unavailable on Android until the foreign
// thread callback path has physical arm64 evidence. Failing before inspecting
// or retaining fn prevents callers from accidentally passing a bogus pointer
// into a Vulkan driver.
func NewCallback(any) uintptr {
	panic("ffi: callbacks are unsupported on Android")
}
