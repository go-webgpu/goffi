#include <stdint.h>
#include <stdarg.h>

// Scalar float returns use XMM0 on Windows AMD64. Keep a pointer argument so
// the end-to-end tests exercise the same call shape as pointer-based native APIs.
float return_float32(const void *value) {
    return value ? 0.125f : 0.0f;
}
double return_float64(const void *value) {
    return value ? 0.625 : 0.0;
}

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

// Mixed int/float structs passed by value. On arm64 a composite that is not
// an HFA goes in X registers as raw bytes, floating-point members included;
// on SysV amd64 each eightbyte is classified on its own.
int64_t take_mixed_int_float(struct mixed_int_float s) {
    return s.a * 1000 + (int64_t)(s.b * 10);
}
int64_t take_mixed_float_int(struct mixed_float_int s) {
    return (int64_t)(s.a * 10) * 1000 + s.b;
}

// {int64, int64}: both INTEGER, returned in RAX:RDX.
struct return_pair_i64 { int64_t a; int64_t b; };
struct return_pair_i64 return_struct_2ints(int64_t a, int64_t b) {
    struct return_pair_i64 s = {.a = a, .b = b};
    return s;
}

// 24-byte struct (> 16 bytes) returned by value through the sret ABI: the caller
// passes a hidden destination pointer (RDI on SysV AMD64, X8 on AAPCS64) and the
// callee writes the struct into it. The Go test calls this with a real rvalue
// buffer and checks the returned fields. triple_i64 is defined above.
struct triple_i64 return_struct_24(void) {
    struct triple_i64 s = {.a = 11, .b = 22, .c = 33};
    return s;
}

// Variadic: sum N int64_t values.
// Prototype: int64_t sum_variadic(int64_t count, ...)
// nfixedargs = 1 (only 'count' is fixed).
int64_t sum_variadic(int64_t count, ...) {
    va_list ap;
    va_start(ap, count);
    int64_t sum = 0;
    for (int64_t i = 0; i < count; i++) {
        sum += va_arg(ap, int64_t);
    }
    va_end(ap);
    return sum;
}

// Variadic: two fixed args plus one variadic int64_t.
// Prototype: int64_t variadic_two_fixed(int64_t a, int64_t b, ...)
// nfixedargs = 2 (a and b are fixed).
int64_t variadic_two_fixed(int64_t a, int64_t b, ...) {
    va_list ap;
    va_start(ap, b);
    int64_t extra = va_arg(ap, int64_t);
    va_end(ap);
    return a + b + extra;
}

// C-to-Go callbacks with floating-point arguments and return values (#93).
// The C compiler is the ground truth for the calling convention: these
// callers place float/double arguments and read float/double results exactly
// as any C library would, so the Go callback sees the real ABI rather than a
// hand-built register frame.

// float arguments arrive in the low 32 bits of XMMn / in Sn.
int32_t cb_call_float_args(int32_t (*f)(float, float), float a, float b) {
    return f(a, b);
}

int32_t cb_call_double_args(int32_t (*f)(double, double), double a, double b) {
    return f(a, b);
}

// float/double results are read from XMM0 / S0 / D0.
float cb_call_float_ret(float (*f)(int64_t), int64_t a) {
    return f(a);
}

double cb_call_double_ret(double (*f)(int64_t), int64_t a) {
    return f(a);
}

// Integer and floating-point arguments interleaved: each class fills its own
// registers independently.
double cb_call_mixed(double (*f)(int32_t, float, int64_t, double, float),
                     int32_t a, float b, int64_t c, double d, float e) {
    return f(a, b, c, d, e);
}

// Nine floats: the first eight in XMM0-7 / S0-S7, the ninth on the stack.
// The result is computed by C from the callback's float return.
typedef float (*float9_fn)(float, float, float, float, float, float, float, float, float);
float cb_call_float9(float9_fn f) {
    return f(1.0f, 2.0f, 3.0f, 4.0f, 5.0f, 6.0f, 7.0f, 8.0f, 9.5f) + 0.25f;
}

// Eight doubles in registers, then a float and a double on the stack.
typedef double (*double8_float_double_fn)(double, double, double, double, double, double, double, double,
                                          float, double);
double cb_call_stack_float_double(double8_float_double_fn f) {
    return f(1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.5f, 10.25);
}

// A callback whose integer result must not be disturbed by the float result
// register: the C caller reads only RAX / X0.
int64_t cb_call_int_ret(int64_t (*f)(double), double a) {
    return f(a);
}

// Stack arguments (#96). Each function weights its arguments by position, so
// an argument read from the wrong stack offset changes the result.

// Ten int32: two are passed on the stack on arm64. Apple packs them into one
// 8-byte slot ([sp], [sp+4]); standard AAPCS64 and SysV use one slot each.
int64_t stack_i32x10(int32_t a0, int32_t a1, int32_t a2, int32_t a3, int32_t a4,
                     int32_t a5, int32_t a6, int32_t a7, int32_t a8, int32_t a9) {
    return a0 + 2 * (int64_t)a1 + 3 * (int64_t)a2 + 4 * (int64_t)a3 + 5 * (int64_t)a4 +
           6 * (int64_t)a5 + 7 * (int64_t)a6 + 8 * (int64_t)a7 + 9 * (int64_t)a8 + 10 * (int64_t)a9;
}

// Ten floats: the ninth and tenth go on the stack.
float stack_f32x10(float a0, float a1, float a2, float a3, float a4,
                   float a5, float a6, float a7, float a8, float a9) {
    return a0 + 2 * a1 + 3 * a2 + 4 * a3 + 5 * a4 + 6 * a5 + 7 * a6 + 8 * a7 + 9 * a8 + 10 * a9;
}

// Eight int64 fill the registers; int8, int16, int32, int8 follow on the stack.
// Apple: offsets 0, 2, 4, 8. Standard AAPCS64 and SysV: 0, 8, 16, 24.
int64_t stack_narrow_mix(int64_t r0, int64_t r1, int64_t r2, int64_t r3,
                         int64_t r4, int64_t r5, int64_t r6, int64_t r7,
                         int8_t a, int16_t b, int32_t c, int8_t d) {
    return r0 + r1 + r2 + r3 + r4 + r5 + r6 + r7 +
           (int64_t)a * 1000000000 + (int64_t)b * 100000 + (int64_t)c * 10 + d;
}

// Eight floats fill the FP registers, so the {float, float} HFA goes on the
// stack as one contiguous 8-byte block (as libffi's allocate_to_stack does).
float stack_hfa_after_8f(float f0, float f1, float f2, float f3,
                         float f4, float f5, float f6, float f7, struct pair_f32 s) {
    return f0 + f1 + f2 + f3 + f4 + f5 + f6 + f7 + s.x * 10 + s.y * 100;
}

// Seven int64 leave one X register; a 16-byte non-HFA struct needs two, so it
// is copied to the stack, not passed by reference (AAPCS64: composites of at
// most 16 bytes go by value; libffi copies them with allocate_to_stack).
int64_t stack_s16_after_7i(int64_t r0, int64_t r1, int64_t r2, int64_t r3,
                           int64_t r4, int64_t r5, int64_t r6, struct pair_i64 s) {
    return r0 + r1 + r2 + r3 + r4 + r5 + r6 + s.a * 1000 + s.b;
}

// Sixteen int64: more arguments than goffi's call path supports on amd64 (15)
// and arm64 (8 registers + 7 stack slots). The call must fail, not drop some.
int64_t stack_i64x16(int64_t a0, int64_t a1, int64_t a2, int64_t a3, int64_t a4, int64_t a5,
                     int64_t a6, int64_t a7, int64_t a8, int64_t a9, int64_t a10, int64_t a11,
                     int64_t a12, int64_t a13, int64_t a14, int64_t a15) {
    return a0 + a1 + a2 + a3 + a4 + a5 + a6 + a7 + a8 + a9 + a10 + a11 + a12 + a13 + a14 + 16 * a15;
}

// Sixteen doubles: eight in FP registers, eight on the stack. That passes the
// generic argument-count check (which assumes 9 stack slots everywhere) but
// exceeds the 7 stack slots of goffi's arm64 call path.
double stack_f64x16(double a0, double a1, double a2, double a3, double a4, double a5,
                    double a6, double a7, double a8, double a9, double a10, double a11,
                    double a12, double a13, double a14, double a15) {
    return a0 + a1 + a2 + a3 + a4 + a5 + a6 + a7 + a8 + a9 + a10 + a11 + a12 + a13 + a14 + 16 * a15;
}

// C `int` arguments and results (#99), described by types.IntTypeDescriptor.
// The weights make a misplaced or misread argument change the result.
int int_weighted3(int a, int b, int c) {
    return a + 2 * b + 3 * c;
}

// Narrow integer arguments widened to 64 bits by the callee: the result shows
// whether the value the caller put in the register or stack slot is extended
// the way the callee's compiler expects. gcc re-extends in the callee; clang
// relies on the caller having extended char and short to 32 bits (Apple clang
// 17, x86-64: narrow_s8 is `movslq %edi, %rax`).
int64_t narrow_s8(int8_t a) { return a; }
int64_t narrow_s16(int16_t a) { return a; }
uint64_t narrow_u8(uint8_t a) { return a; }
uint64_t narrow_u16(uint16_t a) { return a; }

// The same on the stack: eight int64 fill the integer registers on every
// target (6 on SysV, 4 on Win64, 8 on arm64), the narrow arguments follow.
int64_t narrow_stack(int64_t r0, int64_t r1, int64_t r2, int64_t r3, int64_t r4, int64_t r5,
                     int64_t r6, int64_t r7, int8_t a, int16_t b) {
    return r0 + r1 + r2 + r3 + r4 + r5 + r6 + r7 + (int64_t)a * 100000 + b;
}

// Callbacks with stack arguments: C calls the Go callback with 1..10.
int64_t cb_call_stack_i32x10(int64_t (*f)(int32_t, int32_t, int32_t, int32_t, int32_t,
                                          int32_t, int32_t, int32_t, int32_t, int32_t)) {
    return f(1, 2, 3, 4, 5, 6, 7, 8, 9, 10);
}

float cb_call_stack_f32x10(float (*f)(float, float, float, float, float,
                                      float, float, float, float, float)) {
    return f(1.0f, 2.0f, 3.0f, 4.0f, 5.0f, 6.0f, 7.0f, 8.0f, 9.0f, 10.0f);
}

// Structs of N bytes, returned and passed by value. Which sizes travel in
// registers differs per target: Win64 uses a register only for 1, 2, 4 and 8
// bytes and a hidden result pointer / a pointer to a caller-owned copy for every
// other size; System V amd64 and AAPCS64 use registers up to 16 bytes. The Go
// tests check every byte of the result, that nothing past the struct is
// written, and that the callee's writes to its by-value argument never reach
// the caller's value (the volatile stores cannot be optimized away).
#define BYTES_FUNCS(N)                                                    \
    struct bytes##N { uint8_t b[N]; };                                    \
    struct bytes##N return_bytes##N(uint8_t seed) {                       \
        struct bytes##N s;                                                \
        for (int i = 0; i < N; i++) s.b[i] = (uint8_t)(seed + i * 7);     \
        return s;                                                         \
    }                                                                     \
    int64_t take_bytes##N(struct bytes##N s) {                            \
        int64_t sum = 0;                                                  \
        volatile uint8_t *p = s.b;                                        \
        for (int i = 0; i < N; i++) { sum = sum * 31 + p[i]; p[i] = 0xEE; } \
        return sum;                                                       \
    }

BYTES_FUNCS(1)
BYTES_FUNCS(2)
BYTES_FUNCS(3)
BYTES_FUNCS(4)
BYTES_FUNCS(5)
BYTES_FUNCS(6)
BYTES_FUNCS(7)
BYTES_FUNCS(8)
BYTES_FUNCS(9)
BYTES_FUNCS(12)
BYTES_FUNCS(15)
BYTES_FUNCS(16)
BYTES_FUNCS(17)
BYTES_FUNCS(24)
BYTES_FUNCS(32)

// Small structs of floating-point members. System V amd64 returns an eightbyte
// of only float/double members in XMM0 (12 bytes: XMM0:XMM1); AAPCS64 returns
// an HFA of one to four members in S0-S3 / D0-D3, a single-member one included;
// Win64 returns an 8-byte struct in RAX whatever its members, and 12 bytes
// through a hidden result pointer. A mixed eightbyte is INTEGER on System V.
struct one_f32 { float a; };
struct one_f32 return_struct_1float(float a) { struct one_f32 s = {a}; return s; }

struct one_f64 { double a; };
struct one_f64 return_struct_1double(double a) { struct one_f64 s = {a}; return s; }

struct pair_f32 return_struct_2floats(float a, float b) { struct pair_f32 s = {a, b}; return s; }

struct three_f32 { float a, b, c; };
struct three_f32 return_struct_3floats(float a, float b, float c) {
    struct three_f32 s = {a, b, c};
    return s;
}

struct f32_i32 { float a; int32_t b; };
struct f32_i32 return_struct_float_int32(float a, int32_t b) { struct f32_i32 s = {a, b}; return s; }
