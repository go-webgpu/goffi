// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !goffi_static || windows || android

package static

// Enabled reports that this build resolves symbols through the dynamic loader,
// which is the default. On Windows and Android the goffi_static tag is ignored:
// neither platform uses //go:cgo_import_dynamic for library loading.
const Enabled = false
