//go:build goffi_static && (linux || darwin || freebsd || netbsd) && arm64

#include "textflag.h"

// crosscall2 stub for -tags goffi_static.
// Callbacks require fakecgo + dynamic libc; static builds intentionally omit
// that stack. This symbol exists only so the ffi package links; calling it
// aborts.
TEXT crosscall2(SB), NOSPLIT|NOFRAME, $0
	BL runtime·abort(SB)
	RET
