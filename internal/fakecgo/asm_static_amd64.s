//go:build goffi_static && (linux || darwin || freebsd || netbsd) && amd64

#include "textflag.h"

// crosscall2 stub for -tags goffi_static.
// Callbacks require fakecgo + dynamic libc; static builds intentionally omit
// that stack. This symbol exists only so the ffi package links; calling it
// aborts.
TEXT crosscall2(SB), NOSPLIT, $0-0
	CALL runtime·abort(SB)
	RET
