// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build cgo && (linux || darwin || freebsd)

package syscall

// When CGO_ENABLED=1, the syscall package uses runtime.cgocall (via
// //go:linkname) to invoke C functions on a dedicated OS thread. The Go
// runtime only enables cgocall when iscgo is true, which is set by
// runtime/cgo's init. We must therefore drag runtime/cgo into the binary
// from this package, since callers (e.g. internal/arch/arm64) only depend
// on internal/syscall and would otherwise hit "fatal error: cgocall
// unavailable" at first call.
//
// In CGO_ENABLED=0 mode, this file is excluded and internal/fakecgo
// supplies the same machinery.
import _ "runtime/cgo"
