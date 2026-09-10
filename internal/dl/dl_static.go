//go:build goffi_static

// Static-profile stubs for dynamic library loading.
//
// Under -tags goffi_static every //go:cgo_import_dynamic directive is excluded
// so Linux binaries can link fully statically. Dlopen/Dlsym/Dlclose therefore
// return ErrStaticBuild instead of calling libdl.

package dl

import "errors"

// ErrStaticBuild indicates dynamic loading is unavailable in a static build.
var ErrStaticBuild = errors.New("goffi/internal/dl: dynamic loading unavailable in static build (-tags goffi_static)")

// RTLD constants (POSIX values; unused when loading is stubbed).
const (
	RTLD_LAZY   = 0x00001
	RTLD_NOW    = 0x00002
	RTLD_GLOBAL = 0x00100
	RTLD_LOCAL  = 0x00000
)

// RTLD_DEFAULT is a pseudo-handle for dlsym.
const RTLD_DEFAULT = 0x00000

// Dlopen always fails under -tags goffi_static.
func Dlopen(path string, mode int) (uintptr, error) {
	return 0, ErrStaticBuild
}

// Dlsym always fails under -tags goffi_static.
func Dlsym(handle uintptr, name string) (uintptr, error) {
	return 0, ErrStaticBuild
}

// Dlclose always fails under -tags goffi_static.
func Dlclose(handle uintptr) error {
	return ErrStaticBuild
}
