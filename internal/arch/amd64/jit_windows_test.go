// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build amd64 && windows

package amd64

import "github.com/go-webgpu/goffi/types"

const probeMaxArgs = maxWindowsSlots

// probeLocations returns, for each argument, the probe slots that hold it:
// slot i in the i-th integer register, and a float or double in the i-th
// XMM register too, or after the 32-byte home area on the stack.
func probeLocations(cif *types.CallInterface) (locs [][]int, nsse int, fits bool) {
	locs = make([][]int, len(cif.ArgTypes))
	for i, t := range cif.ArgTypes {
		if i >= len(winIntRegs) {
			locs[i] = []int{probeStack + i}
			continue
		}
		locs[i] = []int{probeGPR[winIntRegs[i]]}
		if t.Kind == types.FloatType || t.Kind == types.DoubleType {
			locs[i] = append(locs[i], probeXMM+i)
		}
	}
	return locs, 0, true
}
