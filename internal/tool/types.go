package tool

import "fmt"

// ErrorKind classifies the category of a tool error. The string value is used
// directly as the prefix in tool-result output so the model can distinguish
// error types (e.g. "not_found: ...").
type ErrorKind string

// Sentinel ErrorKind values used by tool implementations to classify OS and
// argument errors into categories the model can act on.
const (
	KindSandboxViolation ErrorKind = "sandbox_violation"
	KindNotFound         ErrorKind = "not_found"
	KindPermissionDenied ErrorKind = "permission_denied"
	KindInvalidArgument  ErrorKind = "invalid_argument"
)

// ToolError is returned by tool implementations when an operation fails in a
// classifiable way. Kind identifies the failure category; Err carries the
// underlying OS or validation error. Implements Unwrap so errors.Is and
// errors.As can traverse the chain.
type ToolError struct {
	Kind ErrorKind
	Err  error
}

func (e *ToolError) Error() string {
	return fmt.Sprintf("%s: %s", e.Kind, e.Err)
}

func (e *ToolError) Unwrap() error {
	return e.Err
}
