// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

// structlib.c — minimal C library for the struct pass/return goffi example.
// Uses integer-only fields so the same binary works on Windows AMD64
// (syscall.SyscallN cannot propagate XMM register values for float struct fields).
#include <stdint.h>

typedef struct { int64_t x; int64_t y; } Point;
typedef struct { int64_t x; int64_t y; int64_t z; } Vec3;

// Return struct by value (16 B — fits two integer GP registers on Unix,
// goes through sret on Windows AMD64).
Point make_point(int64_t x, int64_t y) {
    Point p;
    p.x = x;
    p.y = y;
    return p;
}

// Accept struct by value, return scalar.
int64_t distance_squared(Point a, Point b) {
    int64_t dx = a.x - b.x;
    int64_t dy = a.y - b.y;
    return dx * dx + dy * dy;
}

// Return large struct (24 B — always sret on all platforms).
Vec3 make_vec3(int64_t x, int64_t y, int64_t z) {
    Vec3 v;
    v.x = x;
    v.y = y;
    v.z = z;
    return v;
}

// Accept struct, return scalar.
int64_t vec3_dot(Vec3 a, Vec3 b) {
    return a.x * b.x + a.y * b.y + a.z * b.z;
}
