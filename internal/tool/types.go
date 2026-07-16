package tool

import "fmt"

type ErrorKind string

const (
	KindSandboxViolation ErrorKind = "sandbox_violation"
	KindNotFound         ErrorKind = "not_found"
	KindPermissionDenied ErrorKind = "permission_denied"
)

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
