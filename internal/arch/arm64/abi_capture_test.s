//go:build arm64

#include "textflag.h"

// captureABI stores X0-X7, D0-D7, and X8 into the provided abiCapture buffer.
// The output pointer is expected in X0.
TEXT ·captureABI(SB), NOSPLIT|NOFRAME, $0-8
	MOVD R0, R9

	// GPRs
	MOVD R0, 0(R9)
	MOVD R1, 8(R9)
	MOVD R2, 16(R9)
	MOVD R3, 24(R9)
	MOVD R4, 32(R9)
	MOVD R5, 40(R9)
	MOVD R6, 48(R9)
	MOVD R7, 56(R9)

	// FPRs
	FMOVD F0, 64(R9)
	FMOVD F1, 72(R9)
	FMOVD F2, 80(R9)
	FMOVD F3, 88(R9)
	FMOVD F4, 96(R9)
	FMOVD F5, 104(R9)
	FMOVD F6, 112(R9)
	FMOVD F7, 120(R9)

	// X8 (sret)
	MOVD R8, 128(R9)

	RET
