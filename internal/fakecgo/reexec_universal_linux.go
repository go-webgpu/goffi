// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && linux && !android && goffi_universal

// Portable "universal" re-exec bridge.
//
// A universal goffi binary imports every libc symbol with an empty SONAME, so
// it carries no DT_NEEDED and its PT_INTERP is stripped after linking. That
// makes it loadable by the kernel on any Linux distribution, but it also means
// nothing has mapped a libc into the process: the undefined dlopen/malloc/
// pthread_* symbols are unbound. The first thing that would touch libc is the
// malloc() at the top of x_cgo_init, which runs from rt0_go before the Go
// scheduler, before runtime.args, and before any OS thread is created.
//
// So, at the very top of x_cgo_init, we re-exec the process through the host's
// own dynamic loader with the host libc pre-loaded:
//
//	execve(<host-loader>, {<host-loader>, "--preload", <host-libc-soname>,
//	                       <self>, <original args...>}, <env>+guard)
//
// The host loader maps its libc, the global symbol scope now contains malloc,
// dlopen, pthread_*, __errno_location, and the re-executed process binds them
// from whichever libc the host actually ships (glibc's libc.so.6 or musl's
// libc.musl-<arch>.so.1). A guard variable in the environment stops the second
// launch from re-execing again.
//
// Everything here runs before libc exists, so it uses only raw syscalls
// (rawsyscall6) and mmap'd scratch memory -- never the Go heap, never libc,
// never anything that can grow the stack. String constants live in rodata and
// are copied into an mmap staging buffer with NUL terminators as needed
// (cstr), because C wants NUL-terminated strings and Go string literals are
// not. Argv and envp for the re-exec are read verbatim from /proc/self/cmdline
// and /proc/self/environ (both already NUL-delimited) and merely pointer-ified
// in place.
//
// Failure is best-effort: if no known host loader is found, or execve fails,
// we return and let startup proceed. FFI then cannot work (there is no libc),
// which is the documented limitation of running a universal binary on a system
// whose loader we do not recognise.

package fakecgo

// TLS: on the very first (kernel-direct) launch of a universal binary there is
// no host loader, so the thread pointer is unset (rt0_go delegates TLS setup to
// _cgo_init). setupUniversalTLS, called before this function in x_cgo_init,
// installs a scratch thread pointer so the compiler's g-reloads don't fault;
// see setup_universal_tls_*.s. On the re-executed launch the host loader sets
// up real TLS.
//
// Loader asymmetry (empirically established): musl's ld.so binds the
// empty-SONAME symbols of a PT_INTERP-stripped binary directly, so on musl we
// re-exec our own on-disk binary as-is. glibc's ld.so does NOT bind a
// re-exec'd main object unless it carries a PT_INTERP -- so on glibc we hand
// the loader an in-memory copy (memfd) of the binary with the PT_INTERP header
// restored (restoreInterpToMemfd). The .interp string was left intact when the
// interp was stripped; only the program header's p_type was cleared, so
// restoring it is a single field write. Either way the loader then binds the
// symbols from the pre-loaded libc.

import (
	"unsafe"

	"github.com/go-webgpu/goffi/internal/hostlibc"
)

func rawsyscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1 uintptr)

const (
	atFDCWD = ^uintptr(99) // AT_FDCWD (-100) as unsigned
	oRDONLY = 0
	fOK     = 0
	protRW  = 0x3  // PROT_READ | PROT_WRITE
	mapPA   = 0x22 // MAP_PRIVATE | MAP_ANONYMOUS
	ptrSize = 8    // universal build is 64-bit only

	envBufCap  = 256 << 10
	cmdBufCap  = 256 << 10
	exeBufCap  = 4 << 10
	strBufCap  = 8 << 10
	ptrArrCap  = 4 << 10 // max pointers per argv/envp array (incl. nil terminator)
	ptrArrSize = ptrArrCap * ptrSize

	guardVar = "GOFFI_UNIVERSAL_REEXEC=1"
	guardKey = "GOFFI_UNIVERSAL_REEXEC="

	// What the re-exec destroys, recorded before it happens. Both values are
	// prefixed with the pid they describe, because the environment they live
	// in is inherited by every child: the pid survives execve, so the process
	// the bridge re-execed still matches, and a child (a new pid) does not.
	// ffi.Executable and ffi.Argv0 read these; see ffi/selfinfo.go.
	exeKey   = "GOFFI_UNIVERSAL_EXE="
	argv0Key = "GOFFI_UNIVERSAL_ARGV0="

	// interp restore (glibc path)
	mfdExec   = 0x0010 // MFD_EXEC (kernel 6.3+); fall back to 0 on older kernels
	ptNull    = 0      // PT_NULL  (what the interp header was stripped to)
	ptInterp  = 3      // PT_INTERP
	copyChunk = 64 << 10
)

// Bump allocator over an mmap staging buffer for NUL-terminated C strings.
// Set once, at the start of maybeReexecUniversal; single-threaded at that point.
var (
	strBufBase uintptr // base of mmap staging buffer (uintptr: no GC write barrier)
	strBufOff  uintptr
)

//go:nosplit
func sysErr(r uintptr) bool { return r > ^uintptr(4095) } // r in [-4095, -1]

//go:nosplit
func mmapAnon(n uintptr) unsafe.Pointer {
	r := rawsyscall6(sysMmap, 0, n, protRW, mapPA, ^uintptr(0) /* fd -1 */, 0)
	if sysErr(r) {
		return nil
	}
	return unsafe.Pointer(r)
}

// cstr copies s into the staging buffer, appends a NUL, and returns a *byte to
// the copy. Returns nil if the staging buffer is exhausted.
//
//go:nosplit
func cstr(s string) *byte {
	n := uintptr(len(s))
	if strBufBase == 0 || strBufOff+n+1 > strBufCap {
		return nil
	}
	start := strBufOff
	for i := uintptr(0); i < n; i++ {
		*(*byte)(unsafe.Add(unsafe.Pointer(strBufBase), start+i)) = s[i]
	}
	*(*byte)(unsafe.Add(unsafe.Pointer(strBufBase), start+n)) = 0
	strBufOff = start + n + 1
	return (*byte)(unsafe.Add(unsafe.Pointer(strBufBase), start))
}

// taggedEnv builds "<key><pid>:<val>" NUL-terminated in the staging buffer and
// returns a *byte to it, or nil if val is nil or the buffer is exhausted. val
// is a NUL-terminated C string; its NUL is not copied.
//
//go:nosplit
func taggedEnv(key string, pid uintptr, val *byte) *byte {
	if val == nil || strBufBase == 0 {
		return nil
	}
	var digs [20]byte
	di := len(digs)
	if pid == 0 {
		di--
		digs[di] = '0'
	}
	for v := pid; v > 0; v /= 10 {
		di--
		digs[di] = byte('0' + v%10)
	}
	var valLen uintptr
	for *(*byte)(unsafe.Add(unsafe.Pointer(val), valLen)) != 0 {
		valLen++
	}
	kn := uintptr(len(key))
	dn := uintptr(len(digs) - di)
	if strBufOff+kn+dn+1+valLen+1 > strBufCap {
		return nil
	}
	base := unsafe.Pointer(strBufBase)
	start := strBufOff
	off := start
	for i := uintptr(0); i < kn; i++ {
		*(*byte)(unsafe.Add(base, off)) = key[i]
		off++
	}
	for i := uintptr(0); i < dn; i++ {
		*(*byte)(unsafe.Add(base, off)) = digs[di+int(i)]
		off++
	}
	*(*byte)(unsafe.Add(base, off)) = ':'
	off++
	for i := uintptr(0); i < valLen; i++ {
		*(*byte)(unsafe.Add(base, off)) = *(*byte)(unsafe.Add(unsafe.Pointer(val), i))
		off++
	}
	*(*byte)(unsafe.Add(base, off)) = 0
	strBufOff = off + 1
	return (*byte)(unsafe.Add(base, start))
}

//go:nosplit
func fileExists(pathC *byte) bool {
	if pathC == nil {
		return false
	}
	r := rawsyscall6(sysFaccessat, atFDCWD, uintptr(unsafe.Pointer(pathC)), fOK, 0, 0, 0)
	return r == 0
}

// readAll reads the whole file at pathC into [base, base+capBytes), returning
// the byte count, or -1 on error.
//
//go:nosplit
func readAll(pathC *byte, base unsafe.Pointer, capBytes uintptr) int {
	fd := rawsyscall6(sysOpenat, atFDCWD, uintptr(unsafe.Pointer(pathC)), oRDONLY, 0, 0, 0)
	if sysErr(fd) {
		return -1
	}
	var total uintptr
	for total < capBytes {
		n := rawsyscall6(sysRead, fd, uintptr(unsafe.Add(base, total)), capBytes-total, 0, 0, 0)
		if sysErr(n) {
			rawsyscall6(sysClose, fd, 0, 0, 0, 0, 0)
			return -1
		}
		if n == 0 {
			break
		}
		total += n
	}
	rawsyscall6(sysClose, fd, 0, 0, 0, 0, 0)
	return int(total)
}

//go:nosplit
func setPtr(base unsafe.Pointer, idx int, val uintptr) {
	*(*uintptr)(unsafe.Add(base, uintptr(idx)*ptrSize)) = val
}

// matchAt reports whether the NUL-delimited entry starting at base+off begins
// with key.
//
//go:nosplit
func matchAt(base unsafe.Pointer, off, length int, key string) bool {
	if off+len(key) > length {
		return false
	}
	for i := 0; i < len(key); i++ {
		if *(*byte)(unsafe.Add(base, off+i)) != key[i] {
			return false
		}
	}
	return true
}

// diag writes a short message to stderr (best effort). write(2) takes a
// pointer+length, so no NUL is needed and the message can live in rodata --
// this works even before the cstr staging buffer is set up.
//
//go:nosplit
func diag(msg string) {
	rawsyscall6(sysWrite, 2, uintptr(unsafe.Pointer(unsafe.StringData(msg))), uintptr(len(msg)), 0, 0, 0)
}

// procFdPath writes "/proc/self/fd/<fd>" (NUL-terminated) into the staging
// buffer and returns a *byte to it, or nil if the buffer is exhausted.
//
//go:nosplit
func procFdPath(fd uintptr) *byte {
	const prefix = "/proc/self/fd/"
	var digs [20]byte
	di := len(digs)
	v := fd
	if v == 0 {
		di--
		digs[di] = '0'
	}
	for v > 0 {
		di--
		digs[di] = byte('0' + v%10)
		v /= 10
	}
	pn := uintptr(len(prefix))
	dn := uintptr(len(digs) - di)
	if strBufBase == 0 || strBufOff+pn+dn+1 > strBufCap {
		return nil
	}
	base := unsafe.Pointer(strBufBase)
	start := strBufOff
	for i := uintptr(0); i < pn; i++ {
		*(*byte)(unsafe.Add(base, start+i)) = prefix[i]
	}
	for i := uintptr(0); i < dn; i++ {
		*(*byte)(unsafe.Add(base, start+pn+i)) = digs[di+int(i)]
	}
	*(*byte)(unsafe.Add(base, start+pn+dn)) = 0
	strBufOff = start + pn + dn + 1
	return (*byte)(unsafe.Add(base, start))
}

// patchInterpInBuf finds the stripped PT_INTERP program header in an ELF64
// header buffer (the first chunk of the file) and restores its p_type to
// PT_INTERP. The stripped header is the PT_NULL entry whose p_offset points at
// a path ('/'). Returns false if the header table is not fully present in buf
// or no such entry is found.
//
//go:nosplit
func patchInterpInBuf(buf unsafe.Pointer, n uintptr) bool {
	if n < 64 {
		return false
	}
	phoff := uintptr(*(*uint64)(unsafe.Add(buf, 0x20)))
	phentsize := uintptr(*(*uint16)(unsafe.Add(buf, 0x36)))
	phnum := uintptr(*(*uint16)(unsafe.Add(buf, 0x38)))
	if phentsize < 56 || phoff+phnum*phentsize > n {
		return false
	}
	for i := uintptr(0); i < phnum; i++ {
		pe := phoff + i*phentsize
		if *(*uint32)(unsafe.Add(buf, pe)) != ptNull {
			continue
		}
		poff := uintptr(*(*uint64)(unsafe.Add(buf, pe+8))) // p_offset
		if poff < n && *(*byte)(unsafe.Add(buf, poff)) == '/' {
			*(*uint32)(unsafe.Add(buf, pe)) = ptInterp
			return true
		}
	}
	return false
}

// restoreInterpToMemfd copies /proc/self/exe into a new memfd with the
// PT_INTERP header restored, and returns the memfd descriptor (an -errno-style
// value on failure, testable with sysErr). The memfd is created without
// MFD_CLOEXEC so it survives the upcoming execve and the loader can open it via
// /proc/self/fd/<fd>. Used only on the glibc path.
//
//go:nosplit
func restoreInterpToMemfd() uintptr {
	nameC := cstr("goffi")
	if nameC == nil {
		return ^uintptr(0)
	}
	// No MFD_CLOEXEC: the fd must survive execve so the loader can open
	// /proc/self/fd/<fd>. Prefer MFD_EXEC so the copy may be mapped
	// executable; fall back to 0 on pre-6.3 kernels that reject the flag.
	memfd := rawsyscall6(sysMemfdCreate, uintptr(unsafe.Pointer(nameC)), mfdExec, 0, 0, 0, 0)
	if sysErr(memfd) {
		memfd = rawsyscall6(sysMemfdCreate, uintptr(unsafe.Pointer(nameC)), 0, 0, 0, 0, 0)
		if sysErr(memfd) {
			return ^uintptr(0)
		}
	}
	exeFd := rawsyscall6(sysOpenat, atFDCWD,
		uintptr(unsafe.Pointer(cstr("/proc/self/exe"))), oRDONLY, 0, 0, 0)
	if sysErr(exeFd) {
		rawsyscall6(sysClose, memfd, 0, 0, 0, 0, 0)
		return ^uintptr(0)
	}
	buf := mmapAnon(copyChunk)
	ok := buf != nil && copyExeToMemfd(exeFd, memfd, buf)
	rawsyscall6(sysClose, exeFd, 0, 0, 0, 0, 0)
	if !ok {
		rawsyscall6(sysClose, memfd, 0, 0, 0, 0, 0)
		return ^uintptr(0)
	}
	return memfd
}

// copyExeToMemfd streams exeFd into memfd through buf, restoring the PT_INTERP
// header in the first chunk. Returns false on any short or failed I/O. A plain
// helper (not a closure) so nothing heap-allocates in this pre-scheduler code.
//
//go:nosplit
func copyExeToMemfd(exeFd, memfd uintptr, buf unsafe.Pointer) bool {
	first := true
	for {
		n := rawsyscall6(sysRead, exeFd, uintptr(buf), copyChunk, 0, 0, 0)
		if sysErr(n) {
			return false
		}
		if n == 0 {
			return true
		}
		if first {
			first = false
			if !patchInterpInBuf(buf, n) {
				return false
			}
		}
		for off := uintptr(0); off < n; {
			w := rawsyscall6(sysWrite, memfd, uintptr(unsafe.Add(buf, off)), n-off, 0, 0, 0)
			if sysErr(w) || w == 0 {
				return false
			}
			off += w
		}
	}
}

// maybeReexecUniversal is invoked at the very top of x_cgo_init. On the first
// launch of a universal binary it re-execs through the host loader with libc
// pre-loaded; on the re-executed launch (guard present) it returns immediately.
//
// maybeReexecUniversal is invoked at the very top of x_cgo_init. It either
// hands the process a libc or records that it could not.
//
// Failing to is not fatal by itself: x_cgo_init then drops the process back to
// a pure-Go runtime (see go_linux_{amd64,arm64}.go) and the FFI entry points
// report hostlibc.ErrMissing instead of jumping to unbound symbols. Recording
// it is what makes that possible, so every failure path below has to be seen
// -- hence the split into a function that returns whether a libc is there.
//
//go:nosplit
func maybeReexecUniversal() {
	if !reexecUniversal() {
		hostlibc.Missing = true
	}
}

// reexecUniversal reports whether this process has a libc. It returns true
// only when the guard variable shows we already came through the host loader,
// and false on every path that leaves the empty-SONAME imports unbound. When
// the re-exec succeeds it does not return at all.
//
//go:nosplit
func reexecUniversal() bool {
	// Staging buffer for C strings first: everything below needs cstr().
	sb := mmapAnon(strBufCap)
	if sb == nil {
		return false
	}
	strBufBase = uintptr(sb)
	strBufOff = 0

	envBase := mmapAnon(envBufCap)
	if envBase == nil {
		return false
	}
	envLen := readAll(cstr("/proc/self/environ"), envBase, envBufCap)
	if envLen < 0 {
		return false
	}

	// Guard: if we already re-executed, do nothing.
	for off := 0; off < envLen; {
		if matchAt(envBase, off, envLen, guardKey) {
			return true
		}
		for off < envLen && *(*byte)(unsafe.Add(envBase, off)) != 0 {
			off++
		}
		off++ // skip NUL
	}

	// Pick the host loader + libc SONAME by probing known loader paths.
	// Prefer glibc when both are present; fall back to musl.
	var loaderC, sonameC *byte
	var isGlibc bool
	if g := cstr(glibcLoader); fileExists(g) {
		loaderC = g
		sonameC = cstr(glibcLibc)
		isGlibc = true
	} else if m := cstr(muslLoader); fileExists(m) {
		loaderC = m
		sonameC = cstr(muslLibc)
	}
	if loaderC == nil || sonameC == nil {
		diag("goffi: universal build: no known host dynamic loader found; continuing without FFI\n")
		return false
	}

	// Resolve our own executable path for the loader to run. realExeC keeps
	// that answer even when exeC is replaced by the memfd path below: it is
	// the last moment at which the on-disk location of this binary is
	// knowable, and it is recorded in the environment further down.
	exeBase := mmapAnon(exeBufCap)
	var exeC, realExeC *byte
	if exeBase != nil {
		n := rawsyscall6(sysReadlinkat, atFDCWD,
			uintptr(unsafe.Pointer(cstr("/proc/self/exe"))),
			uintptr(exeBase), exeBufCap-1, 0, 0)
		if !sysErr(n) && n != 0 {
			*(*byte)(unsafe.Add(exeBase, n)) = 0
			exeC = (*byte)(exeBase)
			realExeC = exeC
		}
	}
	if exeC == nil {
		exeC = cstr("/proc/self/exe")
	}

	// glibc only binds a re-exec'd main object that carries a PT_INTERP. Our
	// on-disk binary has none (so the kernel can load it directly on musl), so
	// give glibc an in-memory copy with the interp restored and point the
	// loader at it. musl binds the no-interp binary directly and needs none of
	// this. If the memfd copy fails we fall back to the real path (which will
	// not bind on glibc, but the diagnostic below is best-effort anyway).
	if isGlibc {
		if fd := restoreInterpToMemfd(); !sysErr(fd) {
			if p := procFdPath(fd); p != nil {
				exeC = p
			}
		}
	}

	// Read original argv from /proc/self/cmdline (NUL-delimited).
	cmdBase := mmapAnon(cmdBufCap)
	if cmdBase == nil {
		return false
	}
	cmdLen := readAll(cstr("/proc/self/cmdline"), cmdBase, cmdBufCap)
	if cmdLen < 0 {
		return false
	}

	// Build argv: {loader, "--preload", soname, self, <original argv[1:]...>, NULL}
	argvBase := mmapAnon(ptrArrSize)
	envpBase := mmapAnon(ptrArrSize)
	if argvBase == nil || envpBase == nil {
		return false
	}
	preloadC := cstr("--preload")
	if preloadC == nil {
		return false
	}
	ai := 0
	setPtr(argvBase, ai, uintptr(unsafe.Pointer(loaderC)))
	ai++
	setPtr(argvBase, ai, uintptr(unsafe.Pointer(preloadC)))
	ai++
	setPtr(argvBase, ai, uintptr(unsafe.Pointer(sonameC)))
	ai++
	setPtr(argvBase, ai, uintptr(unsafe.Pointer(exeC)))
	ai++
	// Append original args, skipping argv[0] (the program name) since exeC
	// already occupies argv[0] of the re-executed program.
	{
		off := 0
		// skip argv[0]
		for off < cmdLen && *(*byte)(unsafe.Add(cmdBase, off)) != 0 {
			off++
		}
		off++ // skip its NUL
		for off < cmdLen && ai < ptrArrCap-1 {
			setPtr(argvBase, ai, uintptr(unsafe.Add(cmdBase, off)))
			ai++
			for off < cmdLen && *(*byte)(unsafe.Add(cmdBase, off)) != 0 {
				off++
			}
			off++ // skip NUL
		}
	}
	setPtr(argvBase, ai, 0) // NULL-terminate argv

	// Build envp: <existing environ...> + guard, NULL-terminated.
	// -4: room for the guard, the two recorded variables below, and the NULL.
	ei := 0
	for off := 0; off < envLen && ei < ptrArrCap-4; {
		setPtr(envpBase, ei, uintptr(unsafe.Add(envBase, off)))
		ei++
		for off < envLen && *(*byte)(unsafe.Add(envBase, off)) != 0 {
			off++
		}
		off++ // skip NUL
	}
	if g := cstr(guardVar); g != nil {
		setPtr(envpBase, ei, uintptr(unsafe.Pointer(g)))
		ei++
	}
	// Hand the re-executed process what the re-exec is about to take from it.
	// After the execve, /proc/self/exe is the loader and argv[0] is whatever
	// the loader was told to run -- on glibc a memfd, which names no file at
	// all -- so os.Executable() and os.Args[0] can no longer answer "where am
	// I installed" or "what was I invoked as". Nothing else in the process
	// knows either: this is the only point where both are still true.
	//
	// cmdBase[0] is the original argv[0] (NUL-terminated in place), realExeC
	// the readlink of /proc/self/exe taken above. Either may be absent -- an
	// empty /proc/self/cmdline, a readlink that failed -- and a missing
	// variable is a better answer than a guessed one, so each is recorded
	// only when it is real.
	pid := rawsyscall6(sysGetpid, 0, 0, 0, 0, 0, 0)
	if realExeC != nil && ei < ptrArrCap-2 {
		if v := taggedEnv(exeKey, pid, realExeC); v != nil {
			setPtr(envpBase, ei, uintptr(unsafe.Pointer(v)))
			ei++
		}
	}
	if cmdLen > 0 && *(*byte)(cmdBase) != 0 && ei < ptrArrCap-2 {
		if v := taggedEnv(argv0Key, pid, (*byte)(cmdBase)); v != nil {
			setPtr(envpBase, ei, uintptr(unsafe.Pointer(v)))
			ei++
		}
	}
	setPtr(envpBase, ei, 0) // NULL-terminate envp

	rawsyscall6(sysExecve,
		uintptr(unsafe.Pointer(loaderC)),
		uintptr(argvBase),
		uintptr(envpBase), 0, 0, 0)

	// Only reached if execve failed.
	diag("goffi: universal build: re-exec through host loader failed; continuing without FFI\n")
	return false
}
