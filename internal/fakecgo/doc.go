// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors
// SPDX-FileCopyrightText: 2025-2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && (darwin || freebsd || linux || netbsd)

// Package fakecgo implements the Cgo runtime (runtime/cgo) entirely in Go.
// This allows code that calls into C to function properly when CGO_ENABLED=0.
//
// # Goals
//
// fakecgo attempts to replicate the same naming structure as in the runtime.
// For example, functions that have the prefix "gcc_*" are named "go_*".
// This makes it easier to port other GOOSs and GOARCHs as well as to keep
// it in sync with runtime/cgo.
//
// # Support
//
// Currently, fakecgo supports Linux, macOS, FreeBSD, and NetBSD on amd64 & arm64,
// plus Android arm64/API 29+ as a guarded preview. It cannot be used with
// -buildmode=c-archive because that requires special initialization that fakecgo
// does not implement at the moment.
//
// # Usage
//
// Using fakecgo is easy: just import _ "github.com/go-webgpu/goffi/internal/fakecgo" and then
// set the environment variable CGO_ENABLED=0.
// The recommended usage is to prefer runtime/cgo if possible, but if
// cross-compiling or fast build times are important, fakecgo is available.
// goffi will pick whichever Cgo runtime is available and prefer the one that
// comes with Go (runtime/cgo).
package fakecgo

//go:generate go run gen.go
