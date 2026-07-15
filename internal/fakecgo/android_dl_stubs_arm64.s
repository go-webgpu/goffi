// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build !cgo && android && arm64

#include "textflag.h"

TEXT _android_dlopen(SB), NOSPLIT|NOFRAME, $0-0
	JMP purego_android_dlopen(SB)
	RET

TEXT _android_dlsym(SB), NOSPLIT|NOFRAME, $0-0
	JMP purego_android_dlsym(SB)
	RET
