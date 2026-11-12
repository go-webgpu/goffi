// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build amd64 && (linux || darwin || freebsd)

package amd64

import "github.com/go-webgpu/goffi/types"

const probeMaxArgs = 17

// probeLocations returns, for each argument, the probe slots that hold it,
// the number of vector registers used and whether the stack arguments fit
// in what the probe records.
func probeLocations(cif *types.CallInterface) (locs [][]int, nsse int, fits bool) {
	locs = make([][]int, len(cif.ArgTypes))
	var alloc SysVAllocator
	for i, t := range cif.ArgTypes {
		switch reg, sse, word := alloc.placeScalar(t); {
		case reg < 0:
			if word >= 16 {
				return nil, 0, false
			}
			locs[i] = []int{probeStack + word}
		case sse:
			locs[i] = []int{probeXMM + reg}
		default:
			locs[i] = []int{probeGPR[sysvIntRegs[reg]]}
		}
	}
	return locs, alloc.sse, true
}
