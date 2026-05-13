#include <stdint.h>

// ≤ 8 bytes: {int32, uint32} — INTEGER class, single GP register
struct pair_i32_u32 { int32_t a; uint32_t b; };
int64_t take_struct_8(struct pair_i32_u32 s) {
    return (int64_t)s.a * 1000 + (int64_t)s.b;
}
void callback_struct_8(int32_t a, uint32_t b, void (*callback)(struct pair_i32_u32 s))
{
    struct pair_i32_u32 s = {.a = a, .b = b};
    callback(s);
}

// ≤ 8 bytes: {float, float} — SSE class, single XMM register
struct pair_f32 { float x; float y; };
float take_struct_2floats(struct pair_f32 s) {
    return s.x + s.y;
}
void callback_struct_2floats(float x, float y, void (*callback)(struct pair_f32 s))
{
    struct pair_f32 s = {.x = x, .y = y};
    callback(s);
}

// 16 bytes: {int64, int64} — two INTEGER eightbytes
struct pair_i64 { int64_t a; int64_t b; };
int64_t take_struct_16(struct pair_i64 s) {
    return s.a + s.b;
}
void callback_struct_16(int64_t a, int64_t b, void (*callback)(struct pair_i64 s))
{
    struct pair_i64 s = {.a = a, .b = b};
    callback(s);
}

// 24 bytes: > 16B — MEMORY class, passed on stack
struct triple_i64 { int64_t a; int64_t b; int64_t c; };
int64_t take_struct_24(struct triple_i64 s) {
    return s.a + s.b + s.c;
}
void callback_struct_24(int64_t a, int64_t b, int64_t c, void (*callback)(struct triple_i64 s))
{
    struct triple_i64 s = {.a = a, .b = b, .c = c};
    callback(s);
}

// Mixed: struct arg + scalar args (verify register allocation)
int64_t take_struct_and_int(struct pair_i32_u32 s, int64_t extra) {
    return (int64_t)s.a + (int64_t)s.b + extra;
}
void callback_struct_and_int(int32_t a, uint32_t b, int64_t extra,
                             void (*callback)(struct pair_i32_u32 s, int64_t extra))
{
    struct pair_i32_u32 s = {.a = a, .b = b};
    callback(s, extra);
}

// Struct RETURN functions — test XMM0:XMM1 / RAX:RDX register pair selection.
// {double, double}: SysV AMD64 ABI returns this in XMM0:XMM1 (SSE, SSE).
// Models NSPoint / NSSize on macOS Intel.
struct pair_f64 { double a; double b; };
struct pair_f64 return_struct_2doubles(double a, double b) {
    struct pair_f64 s = {.a = a, .b = b};
    return s;
}

// {int64, double}: eightbyte0 INTEGER (RAX), eightbyte1 SSE (XMM0).
struct mixed_int_float { int64_t a; double b; };
struct mixed_int_float return_struct_int_float(int64_t a, double b) {
    struct mixed_int_float s = {.a = a, .b = b};
    return s;
}

// {double, int64}: eightbyte0 SSE (XMM0), eightbyte1 INTEGER (RAX).
struct mixed_float_int { double a; int64_t b; };
struct mixed_float_int return_struct_float_int(double a, int64_t b) {
    struct mixed_float_int s = {.a = a, .b = b};
    return s;
}

// {int64, int64}: both INTEGER, returned in RAX:RDX.
struct return_pair_i64 { int64_t a; int64_t b; };
struct return_pair_i64 return_struct_2ints(int64_t a, int64_t b) {
    struct return_pair_i64 s = {.a = a, .b = b};
    return s;
}
