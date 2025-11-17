//go:build linux && amd64

#include "textflag.h"

// JMP stubs to dynamically linked symbols
// These symbols are linked via //go:cgo_import_dynamic in dl_linux_nocgo.go

// dlopen_stub: JMP to dlopen from libdl.so.2
TEXT dlopen_stub(SB), NOSPLIT|NOFRAME, $0-0
	JMP goffi_dlopen(SB)

// dlsym_stub: JMP to dlsym from libdl.so.2
TEXT dlsym_stub(SB), NOSPLIT|NOFRAME, $0-0
	JMP goffi_dlsym(SB)

// dlerror_stub: JMP to dlerror from libdl.so.2
TEXT dlerror_stub(SB), NOSPLIT|NOFRAME, $0-0
	JMP goffi_dlerror(SB)
