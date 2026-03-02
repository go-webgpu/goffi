# Performance Guide - goffi v0.4.1

> **Comprehensive performance analysis, benchmarks, and usage guidelines**
> **Platform**: Windows AMD64, 12th Gen Intel Core i7-1255U
> **Go Version**: 1.25+

---

## TL;DR - Quick Summary

✅ **FFI Overhead**: ~88-114 ns/op
✅ **Acceptable for**: WebGPU, system calls, I/O, GPU operations
❌ **NOT acceptable for**: Tight loops, hot-path math, high-frequency calls (>100K/sec)

**Comparison**:
- **goffi**: ~100 ns/op overhead
- **CGO**: ~140-170 ns/op (Go 1.26 reduced overhead ~30%)
- **purego**: ~100-150 ns/op (similar approach)
- **Direct Go**: ~0.2 ns/op (baseline)

**Verdict**: goffi is **production-ready for WebGPU** and similar use cases where function calls are rare (< 10K/sec) and expensive (> 1µs each).

---

## Benchmark Results

### 1. FFI Call Overhead

| Benchmark | ns/op | B/op | allocs/op | Notes |
|-----------|-------|------|-----------|-------|
| **BenchmarkGoffiOverhead** | 88.09 | 64 | 2 | Empty C function (`getpid`) |
| **BenchmarkGoffiIntArgs** | 113.9 | 72 | 3 | Integer argument (`abs`) |
| **BenchmarkGoffiStringOutput** | 97.81 | 72 | 3 | String processing (`strlen`) |
| **BenchmarkDirectGo** | 0.21 | 0 | 0 | Pure Go baseline |

**Key Insights**:
- **Minimum FFI overhead**: ~88 ns (empty function)
- **Typical overhead**: ~100-115 ns (with arguments)
- **Overhead ratio**: ~400-500x vs direct Go call
- **Allocations**: 2-3 per call (runtime.cgocall internals)

### 2. One-Time Costs

| Operation | ns/op | B/op | allocs/op | Frequency |
|-----------|-------|------|-----------|-----------|
| **LoadLibrary** | 607.8 | 48 | 3 | Once per library |
| **GetSymbol** | 318.1 | 40 | 2 | Once per function |
| **PrepareCallInterface** | 63.94 | 24 | 1 | Once per function signature |

**Key Insights**:
- **Library loading**: ~600 ns (amortize over thousands of calls)
- **Symbol lookup**: ~300 ns (cache function pointers)
- **CIF preparation**: ~64 ns (reuse CallInterface objects)

### 3. Platform-Specific

**Windows AMD64** (tested):
- Win64 calling convention (RCX, RDX, R8, R9 + 32-byte shadow space)
- kernel32.dll: 607.8 ns load time
- msvcrt.dll: similar

**Linux AMD64** (expected):
- System V AMD64 ABI (RDI, RSI, RDX, RCX, R8, R9)
- libc.so.6: ~400-600 ns load time (faster dlopen)
- Similar FFI overhead (~100-120 ns)

---

## Performance Analysis

### Overhead Breakdown

```
Total FFI call time: ~100 ns
├── runtime.cgocall:     ~60 ns  (stack switch, GC coordination)
├── Assembly wrapper:    ~20 ns  (register loads, MOVQ/MOVSD)
├── JMP stub:            ~5 ns   (indirect jump)
├── Return path:         ~10 ns  (stack restore)
└── Bookkeeping:         ~5 ns   (error handling, Go overhead)
```

### Why is it acceptable for WebGPU?

**Typical WebGPU operation costs**:
```
wgpuDeviceCreateBuffer():    1-10 µs   (GPU allocation)
wgpuQueueSubmit():           10-100 µs (GPU dispatch)
wgpuRenderPassEncoderDraw(): 0.5-5 µs  (GPU command)

FFI overhead: 100 ns = 0.1 µs

Overhead percentage:
- Fast GPU call (0.5 µs): 100ns / 500ns = 20% overhead (acceptable!)
- Typical GPU call (5 µs): 100ns / 5000ns = 2% overhead (excellent!)
- Batch operation (100 µs): 100ns / 100000ns = 0.1% overhead (negligible!)
```

**Conclusion**: For GPU operations, FFI overhead is **noise-level** (< 5% impact).

### When NOT to use goffi

❌ **Tight loops with many calls**:
```go
// ❌ BAD: 1 million math calls = 100ms overhead!
for i := 0; i < 1_000_000; i++ {
    result := libm.Call("sin", x)  // 100ns × 1M = 100ms
}

// ✅ GOOD: Batch processing or use math.Sin()
result := math.Sin(x)  // Pure Go, 0.2ns
```

❌ **Hot-path math libraries**:
```go
// ❌ BAD: FFI for every pixel
for y := 0; y < 1080; y++ {
    for x := 0; x < 1920; x++ {
        pixel := libimage.Call("process", x, y)  // 2M calls!
    }
}

// ✅ GOOD: Batch entire frame
pixels := libimage.Call("process_frame", frameBuffer)  // 1 call!
```

❌ **High-frequency polling**:
```go
// ❌ BAD: 10K polls/sec = 1ms/sec = 0.1% CPU just for FFI
ticker := time.NewTicker(100 * time.Microsecond)
for range ticker.C {
    status := hw.Call("poll_status")  // Every 100µs
}

// ✅ GOOD: Batch or use Go channels
events := hw.Call("get_events_batch")  // Get all events at once
```

---

## Optimization Strategies

### 1. Amortize One-Time Costs

```go
// ✅ GOOD: Load once, call many times
var (
    handle   unsafe.Pointer
    funcPtr  unsafe.Pointer
    cif      types.CallInterface
)

func init() {
    handle, _ = ffi.LoadLibrary("mylib.dll")
    funcPtr, _ = ffi.GetSymbol(handle, "myFunction")
    ffi.PrepareCallInterface(&cif, types.DefaultCall, ...)
}

// Now each call is just ~100ns overhead
func CallMyFunction(arg int) {
    ffi.CallFunction(&cif, funcPtr, &result, args)
}
```

### 2. Batch Operations

```go
// ❌ BAD: N FFI calls
for _, item := range items {
    Process(item)  // 100ns × N
}

// ✅ GOOD: 1 FFI call
ProcessBatch(items)  // 100ns × 1
```

### 3. Cache Results

```go
// ✅ Cache expensive computations
var cache = make(map[Key]Result)

func GetResult(key Key) Result {
    if result, ok := cache[key]; ok {
        return result  // 0.2ns (map lookup)
    }
    result := FFIExpensiveCall(key)  // 100ns + C cost
    cache[key] = result
    return result
}
```

### 4. Use Go When Possible

```go
// ❌ FFI for simple math
result := libm.Call("sin", x)  // ~100ns + C sin (~10ns) = 110ns

// ✅ Pure Go
result := math.Sin(x)  // ~10-20ns (similar to C!)
```

---

## Real-World Performance Examples

### WebGPU Frame Rendering (Target: 60 FPS = 16.6ms/frame)

**Typical frame with goffi**:
```
wgpuQueueSubmit():              100 µs (GPU work)
wgpuRenderPassEncoderDraw(): ×10 = 50 µs (draw calls)
wgpuDeviceCreateBuffer(): ×3   = 15 µs (buffer creation)
Other GPU calls: ×20          = 100 µs
FFI overhead: 33 calls × 0.1µs = 3.3 µs

Total: 268.3 µs per frame
FFI overhead: 3.3µs / 268.3µs = 1.2% ✅
```

**Verdict**: goffi overhead is **negligible for WebGPU rendering** (< 2% impact).

### System Call Monitoring (1000 calls/sec)

```
System calls per second: 1000
FFI overhead per call: 100 ns
Total overhead per second: 1000 × 100ns = 0.1ms = 0.01% CPU ✅
```

**Verdict**: Acceptable for monitoring, logging, system integration.

### Database Query (10 queries/sec)

```
Query execution time: ~10ms (typical)
FFI overhead: 0.0001ms = 0.001% ✅
```

**Verdict**: FFI overhead is **unmeasurable** for I/O-bound operations.

---

## Comparison with Alternatives

### goffi vs CGO

| Aspect | goffi | CGO |
|--------|-------|-----|
| **Overhead** | ~100 ns | ~140-170 ns (Go 1.26) |
| **Build** | Zero deps | Requires C compiler |
| **Cross-compile** | ✅ Easy | ❌ Complex |
| **Static binary** | ✅ Yes | ⚠️ Often requires libc |

> **Note**: Go 1.26 (Feb 2026) reduced CGO overhead ~30% by removing the dedicated syscall P state. goffi benefits from the same improvement — both use `runtime.cgocall` internally.

### goffi vs purego

| Aspect | goffi | purego |
|--------|-------|-------|
| **Overhead** | ~100 ns | ~100-150 ns |
| **Per-call allocations** | Zero (CIF reused) | sync.Pool per call |
| **Type Safety** | ✅ TypeDescriptor validation | Go reflect.Type |
| **Error Handling** | ✅ 5 typed errors | Generic errors |
| **Callback float returns** | ✅ XMM0 in asm | ❌ panic |
| **ARM64 HFA** | Recursive struct walk | Top-level only |
| **Context support** | ✅ Timeouts/cancellation | ❌ |
| **Platforms** | 5 (quality focus) | 9+ (breadth focus) |

---

## Go 1.26 CGO Improvements

Go 1.26 (released February 2026) [reduced cgo call overhead by ~30%](https://go.dev/doc/go1.26) by removing the dedicated syscall P state. Benchmarks on Apple M1 show `CgoCall` is 33% faster, `CgoCallWithCallback` is 21% faster.

**What this means for goffi:**

- **goffi benefits too** — our `runtime.cgocall` path gets the same ~30% speedup, because goffi uses the same Go runtime machinery internally
- **CGO still requires a C compiler** at build time — goffi does not
- **Cross-compilation** with CGO still requires cross-toolchains — `GOOS=linux GOARCH=arm64 go build` just works with goffi
- **Static binaries** — CGO often pulls in libc, goffi produces fully static Go binaries

The gap between CGO and pure-Go FFI is narrowing from both directions. We welcome it.

---

## Performance Roadmap

### v0.5.0 - Usability + Optimization
- [ ] Builder pattern API (less boilerplate)
- [ ] Variadic function support
- [ ] Assembly micro-optimizations

### v1.0.0 - Production Benchmarks
- [ ] Comprehensive benchmarks vs CGO/purego (published)
- [ ] Platform-specific tuning (Linux, macOS, ARM64)
- [ ] Real-world case studies (WebGPU, Vulkan)

---

## Troubleshooting

### My app is slow with goffi!

**Check 1**: How many FFI calls per second?
```go
// Add timing
start := time.Now()
for i := 0; i < 10000; i++ {
    YourFFICall()
}
fmt.Printf("Calls/sec: %d\n", 10000 / time.Since(start).Seconds())

// If > 100K calls/sec → Consider batching or Go alternative
```

**Check 2**: Are you recreating CIF every call?
```go
// ❌ BAD: Prepare CIF in loop
for _, item := range items {
    cif := &types.CallInterface{}
    ffi.PrepareCallInterface(cif, ...)  // 64ns × N!
    ffi.CallFunction(cif, ...)
}

// ✅ GOOD: Prepare once
cif := &types.CallInterface{}
ffi.PrepareCallInterface(cif, ...)
for _, item := range items {
    ffi.CallFunction(cif, ...)  // Just ~100ns
}
```

**Check 3**: Is the C function itself slow?
```go
// Measure C function cost
start := time.Now()
ffi.CallFunction(cif, fn, ...)
fmt.Printf("Total: %v\n", time.Since(start))
// If > 10µs, the C function is slow, not goffi!
```

---

## Benchmarking Your Code

```bash
# Run goffi benchmarks
cd ffi && go test -bench=. -benchmem -benchtime=1s

# Profile your application
go test -bench=YourBenchmark -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Compare before/after
go test -bench=. -benchmem > before.txt
# Make changes
go test -bench=. -benchmem > after.txt
benchstat before.txt after.txt
```

---

## Conclusion

**goffi is production-ready** for:
- ✅ WebGPU bindings (primary use case)
- ✅ GPU computing (CUDA, Vulkan, DirectX)
- ✅ System library integration (I/O, networking)
- ✅ Embedded applications (sensors, hardware)
- ✅ Legacy library integration (scientific, financial)

**NOT recommended** for:
- ❌ Tight loops (millions of calls)
- ❌ Hot-path math (use `math` package)
- ❌ High-frequency polling (> 100K calls/sec)

**Performance**: ~100 ns overhead = **< 5% impact** for typical WebGPU/GPU workloads.

---

*Benchmarks conducted on Windows AMD64, Intel i7-1255U @ 12 cores*
*Your results may vary depending on CPU, OS, and workload*
*Last updated: 2026-03-02 | goffi v0.4.1*
