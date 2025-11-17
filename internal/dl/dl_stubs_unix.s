//go:build (linux || darwin) && amd64

#include "textflag.h"

// JMP stubs to dynamically linked symbols
// These symbols are linked via //go:cgo_import_dynamic in:
//   - dl_linux_nocgo.go (Linux: libdl.so.2)
//   - dl_darwin_nocgo.go (macOS: libSystem.B.dylib)

// dlopen_stub: JMP to dlopen
TEXT dlopen_stub(SB), NOSPLIT|NOFRAME, $0-0
	JMP goffi_dlopen(SB)

// dlsym_stub: JMP to dlsym
TEXT dlsym_stub(SB), NOSPLIT|NOFRAME, $0-0
	JMP goffi_dlsym(SB)

// dlerror_stub: JMP to dlerror
TEXT dlerror_stub(SB), NOSPLIT|NOFRAME, $0-0
	JMP goffi_dlerror(SB)
