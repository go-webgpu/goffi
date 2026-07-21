// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build android && arm64

package ffi

import "testing"

func TestNewCallbackFailsExplicitlyOnAndroid(t *testing.T) {
	defer func() {
		got := recover()
		if got != "ffi: callbacks are unsupported on Android" {
			t.Fatalf("NewCallback panic = %v, want stable unsupported message", got)
		}
	}()
	NewCallback(func() {})
}
