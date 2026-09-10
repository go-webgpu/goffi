// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

// Command goffi-strip-interp removes the ELF program interpreter (PT_INTERP)
// from a binary produced with -tags goffi_universal.
//
// The Go linker always writes the default glibc interpreter path, which does
// not exist on a musl-only system, so the kernel would refuse to exec the
// binary there. Flipping the PT_INTERP program header to PT_NULL makes the
// kernel load the binary directly on every distribution (as it does for fully
// static binaries). goffi's universal build then brings up libc itself, by
// re-executing through the host's own loader -- see docs/PROFILE_U.md.
//
// This is deliberately a tiny, self-contained ELF edit (no external tools):
// find the PT_INTERP entry in the program header table and zero its p_type.
// Nothing else in the file is touched; the (now unreferenced) .interp bytes
// are harmless.
//
// Usage:
//
//	goffi-strip-interp <binary> [<binary>...]
package main

import (
	"debug/elf"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: goffi-strip-interp <binary> [<binary>...]")
		os.Exit(2)
	}
	status := 0
	for _, path := range os.Args[1:] {
		if err := stripInterp(path); err != nil {
			fmt.Fprintf(os.Stderr, "goffi-strip-interp: %s: %v\n", path, err)
			status = 1
			continue
		}
		fmt.Printf("goffi-strip-interp: %s: PT_INTERP removed\n", path)
	}
	os.Exit(status)
}

func stripInterp(path string) (err error) {
	// Read enough of the ELF header to locate the program header table.
	f, err := elf.Open(path)
	if err != nil {
		return err
	}
	class := f.Class
	byteOrder := f.ByteOrder
	interpIdx := -1
	for i, p := range f.Progs {
		if p.Type == elf.PT_INTERP {
			interpIdx = i
			break
		}
	}
	_ = f.Close()

	if interpIdx < 0 {
		return nil // already interpreter-less; nothing to do
	}

	fh, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := fh.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// Re-read the raw ELF header fields we need. Offsets are fixed by the ELF
	// spec and differ between the 32- and 64-bit forms.
	hdr := make([]byte, 64)
	if _, err := io.ReadFull(fh, hdr); err != nil {
		return err
	}
	var phoff int64
	var phentsize, phnum int
	switch class {
	case elf.ELFCLASS64:
		phoff = int64(byteOrder.Uint64(hdr[0x20:]))
		phentsize = int(byteOrder.Uint16(hdr[0x36:]))
		phnum = int(byteOrder.Uint16(hdr[0x38:]))
	case elf.ELFCLASS32:
		phoff = int64(byteOrder.Uint32(hdr[0x1c:]))
		phentsize = int(byteOrder.Uint16(hdr[0x2a:]))
		phnum = int(byteOrder.Uint16(hdr[0x2c:]))
	default:
		return fmt.Errorf("unsupported ELF class %v", class)
	}
	if interpIdx >= phnum {
		return fmt.Errorf("PT_INTERP index %d out of range (phnum=%d)", interpIdx, phnum)
	}

	// p_type is the first 4 bytes of every program header entry, in both the
	// 32- and 64-bit layouts. Overwrite it with PT_NULL (0).
	off := phoff + int64(interpIdx)*int64(phentsize)
	zero := make([]byte, 4)
	binary.LittleEndian.PutUint32(zero, uint32(elf.PT_NULL)) // 0; endianness irrelevant for 0
	if _, err := fh.WriteAt(zero, off); err != nil {
		return err
	}
	return nil
}
