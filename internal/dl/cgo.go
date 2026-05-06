// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build cgo && (darwin || freebsd || linux || netbsd)

package dl

// When CGO_ENABLED=1, drag in runtime/cgo so that the Go runtime properly
// initializes its cgo machinery. The dl wrappers below use runtime.cgocall
// (via //go:linkname) which only works when iscgo is true; runtime/cgo's
// init is what flips that flag.
//
// Some frameworks also need TLS to be set up the C way, which Go does not
// do unless runtime/cgo is linked. Even with CGO_ENABLED=1, runtime/cgo is
// not pulled in unless `import "C"` is used somewhere. Goffi never uses
// `import "C"`, so we have to do the import here ourselves.
//
// In CGO_ENABLED=0 mode, this file is excluded and internal/fakecgo (kept
// in lockstep with this build tag via `//go:build !cgo`) supplies the
// equivalent runtime symbols.
import _ "runtime/cgo"
