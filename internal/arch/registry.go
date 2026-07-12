package arch

import (
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// FunctionCaller defines the contract for function execution.
type FunctionCaller interface {
	Execute(cif *types.CallInterface, fn unsafe.Pointer, rvalue unsafe.Pointer, avalue []unsafe.Pointer) error
}

// FunctionCallerErrno is an optional extension of FunctionCaller for platforms
// that support in-trampoline errno capture. Callers should check for this
// interface via type assertion before calling ExecuteErrno.
type FunctionCallerErrno interface {
	// ExecuteErrno is like Execute but also captures the C errno value set by
	// the called function. The errnoFn parameter is the address of
	// __errno_location (Linux/FreeBSD) or __error (macOS), obtained from
	// internal/syscall.ErrnoFnAddr(). When errnoFn is 0, cerrno is always 0.
	ExecuteErrno(cif *types.CallInterface, fn unsafe.Pointer, rvalue unsafe.Pointer, avalue []unsafe.Pointer, errnoFn uintptr) (cerrno uintptr, err error)
}

// ArgumentClassifier defines the contract for argument classification.
type ArgumentClassifier interface {
	ClassifyReturn(t *types.TypeDescriptor, abi types.CallingConvention) int
	ClassifyArgument(t *types.TypeDescriptor, abi types.CallingConvention) ArgumentClassification
}

// ArgumentClassification contains argument passing information.
type ArgumentClassification struct {
	GPRCount int
	SSECount int
}

// Registry contains registered implementations.
var Registry struct {
	Caller     FunctionCaller
	Classifier ArgumentClassifier
}

// Register registers implementations for the current architecture.
func Register(caller FunctionCaller, classifier ArgumentClassifier) {
	Registry.Caller = caller
	Registry.Classifier = classifier
}
