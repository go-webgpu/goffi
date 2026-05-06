#include <stdint.h>

// ≤ 8 bytes: {int32, uint32} — INTEGER class, single GP register
struct pair_i32_u32 { int32_t a; uint32_t b; };
int64_t take_struct_8(struct pair_i32_u32 s) {
    return (int64_t)s.a * 1000 + (int64_t)s.b;
}

// ≤ 8 bytes: {float, float} — SSE class, single XMM register
struct pair_f32 { float x; float y; };
float take_struct_2floats(struct pair_f32 s) {
    return s.x + s.y;
}

// 16 bytes: {int64, int64} — two INTEGER eightbytes
struct pair_i64 { int64_t a; int64_t b; };
int64_t take_struct_16(struct pair_i64 s) {
    return s.a + s.b;
}

// 24 bytes: > 16B — MEMORY class, passed on stack
struct triple_i64 { int64_t a; int64_t b; int64_t c; };
int64_t take_struct_24(struct triple_i64 s) {
    return s.a + s.b + s.c;
}

// Mixed: struct arg + scalar args (verify register allocation)
int64_t take_struct_and_int(struct pair_i32_u32 s, int64_t extra) {
    return (int64_t)s.a + (int64_t)s.b + extra;
}
