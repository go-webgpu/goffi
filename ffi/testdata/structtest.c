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
