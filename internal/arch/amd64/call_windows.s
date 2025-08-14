//go:build amd64 && windows

#include "textflag.h"

// func callWin64(gpr []uint64, sse []float64, fn unsafe.Pointer) uint64
TEXT ·callWin64(SB), NOSPLIT, $32-56  // 32 байта shadow space
    // Загрузка аргументов Go
    MOVQ gpr+0(FP), AX
    MOVQ sse+24(FP), BX
    MOVQ fn+48(FP), R11

    // Загрузка регистров для вызова
    MOVQ 0(AX), CX
    MOVQ 8(AX), DX
    MOVQ 16(AX), R8
    MOVQ 24(AX), R9
    MOVSD 0(BX), X0
    MOVSD 8(BX), X1
    MOVSD 16(BX), X2
    MOVSD 24(BX), X3

    // Вызов функции
    CALL R11

    // Сохранение результата
    MOVQ AX, ret+56(FP)
    RET
