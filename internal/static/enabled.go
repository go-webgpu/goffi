// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build goffi_static && !windows && !android

package static

// Enabled reports that this build has no dynamic symbol imports and therefore
// cannot load shared libraries or call foreign functions.
const Enabled = true
