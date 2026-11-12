// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !((((linux && !android) || (darwin && !ios)) && (amd64 || arm64)) || (windows && amd64))

package jit

import "errors"

const supported = false

const chunkHeader = 0

func validCode([]byte) bool { return false }

func allocExec(int) ([]byte, error) { return nil, errors.ErrUnsupported }

func sealExec(*chunk) error { return errors.ErrUnsupported }
