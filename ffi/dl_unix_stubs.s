//go:build linux && amd64

#include "textflag.h"

// Assembly stubs that jump to dynamically imported libc functions

// func dlopen(path *byte, mode int) uintptr
TEXT ·dlopen(SB), NOSPLIT|NOFRAME, $0-0
	JMP libc_dlopen(SB)

// func dlsym(handle uintptr, symbol *byte) uintptr
TEXT ·dlsym(SB), NOSPLIT|NOFRAME, $0-0
	JMP libc_dlsym(SB)

// func dlerror() *byte
TEXT ·dlerror(SB), NOSPLIT|NOFRAME, $0-0
	JMP libc_dlerror(SB)
