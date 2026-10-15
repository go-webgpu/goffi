// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build netbsd && !cgo && !goffi_static

package fakecgo

import _ "unsafe" // for go:linkname

// Supply environ, __progname and __ps_strings, because we don't
// link against the standard NetBSD crt0.o and the
// libc dynamic library needs them.
//
// The three symbols must land in the *dynamic* symbol table, not just the
// static one: NetBSD's libc.so carries undefined references to all three
// (crt0 normally defines them) and rtld resolves them against the main object
// at startup. A plain //go:linkname only produces a local definition, so the
// process would die in rtld before reaching main. //go:cgo_export_dynamic is
// what promotes them, exactly as freebsd.go does.
//
// Note: when cross-compiling or building with CGO_ENABLED=0, add
// the following argument to `go` so that these symbols are defined by
// making fakecgo the Cgo.
//
//	-gcflags="github.com/go-webgpu/goffi/internal/fakecgo=-std"

//go:linkname _environ environ
//go:linkname _progname __progname
//go:linkname ___ps_strings __ps_strings

//go:cgo_export_dynamic environ
//go:cgo_export_dynamic __progname
//go:cgo_export_dynamic __ps_strings

var (
	_environ      uintptr
	_progname     uintptr
	___ps_strings uintptr
)
