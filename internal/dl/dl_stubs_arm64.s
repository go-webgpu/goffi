//go:build ((linux && !android && !cgo) || darwin || freebsd) && arm64

#include "textflag.h"

// JMP stubs to dynamically linked symbols (ARM64)
// These symbols are linked via //go:cgo_import_dynamic in:
//   - dl_linux.go (Linux: libdl.so.2)
//   - dl_darwin.go (macOS: libSystem.B.dylib)

// dlopen_stub: B to dlopen
TEXT dlopen_stub(SB), NOSPLIT|NOFRAME, $0-0
	B goffi_dlopen(SB)

// dlsym_stub: B to dlsym
TEXT dlsym_stub(SB), NOSPLIT|NOFRAME, $0-0
	B goffi_dlsym(SB)

// dlerror_stub: B to dlerror
TEXT dlerror_stub(SB), NOSPLIT|NOFRAME, $0-0
	B goffi_dlerror(SB)
