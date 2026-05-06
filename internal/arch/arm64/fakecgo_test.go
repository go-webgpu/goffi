// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build arm64 && !cgo

package arm64

// When CGO_ENABLED=0, the runtime.cgocall machinery used by the FFI
// implementation is supplied by internal/fakecgo. When CGO_ENABLED=1, the
// standard runtime/cgo package is linked transparently via internal/syscall
// (and internal/dl), so this import must be excluded to avoid pulling a
// package whose build constraints exclude every Go file.
import _ "github.com/go-webgpu/goffi/internal/fakecgo"
