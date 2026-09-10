// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

// Command goffi-audit checks that a binary satisfies the "Profile U" ELF
// contract: no PT_INTERP program header and no DT_NEEDED dynamic entries. It is
// the portable, debug/elf-based equivalent of static-everywhere's onebin
// profile check. Exit status is non-zero if any listed binary fails.
//
// Usage: goffi-audit <binary> [binary...]
package main

import (
	"debug/elf"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: goffi-audit <binary> [binary...]")
		os.Exit(2)
	}
	rc := 0
	for _, path := range os.Args[1:] {
		if err := audit(path); err != nil {
			fmt.Printf("FAIL %s: %v\n", path, err)
			rc = 1
		} else {
			fmt.Printf("ok   %s: no PT_INTERP, no DT_NEEDED\n", path)
		}
	}
	os.Exit(rc)
}

func audit(path string) error {
	f, err := elf.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	var problems []string
	for _, p := range f.Progs {
		if p.Type == elf.PT_INTERP {
			problems = append(problems, "has PT_INTERP")
			break
		}
	}
	if libs, err := f.ImportedLibraries(); err != nil {
		return err
	} else if len(libs) > 0 {
		problems = append(problems, "has DT_NEEDED: "+strings.Join(libs, ", "))
	}
	if len(problems) > 0 {
		return fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	return nil
}
