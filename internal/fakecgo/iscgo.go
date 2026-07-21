// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !cgo && (darwin || freebsd || linux || netbsd)

// The runtime package contains an uninitialized definition
// for runtime·iscgo. Override it to tell the runtime we're here.
// There are various function pointers that should be set too,
// but those depend on dynamic linker magic to get initialized
// correctly, and sometimes they break. This variable is a
// backup: it depends only on old C style static linking rules.
//
// Android builds also satisfy the linux build term, so this declaration is
// intentionally selected for Android/arm64. Outbound runtime.cgocall requires
// iscgo even though goffi rejects Android C-to-Go callbacks at its public API.

package fakecgo

import _ "unsafe" // for go:linkname

//go:linkname _iscgo runtime.iscgo
var _iscgo bool = true
