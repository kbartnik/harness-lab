// Package tool defines the Tool interface and provides Read and Write
// implementations that operate within a sandboxed directory. All file access
// is restricted to a configurable root path; attempts to escape it are
// returned to the model as sandbox_violation errors rather than hard failures.
package tool

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kbartnik/harness-lab/internal/core"
)

// Tool is the interface that all agent tools must implement. Name and
// Description are sent to the model so it knows what tools are available and
// when to use them. Schema returns a JSON Schema object describing the expected
// arguments. Execute runs the tool and returns a ToolResult; errors returned
// from Execute are hard failures — handled errors (file not found, etc.) should
// be returned as ToolResult with IsError set instead.
type Tool interface {
	// Name returns the tool's identifier as sent to the model.
	Name() string
	// Description returns a human-readable summary of what the tool does.
	Description() string
	// Schema returns a JSON Schema object describing the tool's input arguments.
	Schema() map[string]any
	// Execute runs the tool with the provided arguments.
	Execute(args map[string]any) (core.ToolResult, error)
}

// errorResult constructs a ToolResult that signals a handled error to the
// model. Tools use this instead of returning a Go error so the model receives
// the failure description and can decide how to proceed.
func errorResult(msg string) core.ToolResult {
	return core.ToolResult{IsError: true, Output: msg}
}

// fileErrorResult converts an OS file error into a ToolResult. It tries to
// classify the error into a known ErrorKind first; if that fails it falls back
// to the raw error message so no information is lost.
func fileErrorResult(err error) core.ToolResult {
	if toolErr := classifyFileError(err); toolErr != nil {
		return errorResult(toolErr.Error())
	}
	return errorResult(err.Error())
}

// classifyFileError maps a raw OS error to a ToolError with the appropriate
// ErrorKind. Returns nil if the error doesn't match any known category, which
// callers treat as a fallback to the raw message.
func classifyFileError(err error) *ToolError {
	switch {
	case os.IsNotExist(err):
		return &ToolError{KindNotFound, err}
	case os.IsPermission(err):
		return &ToolError{KindPermissionDenied, err}
	default:
		return nil
	}
}

// resolveInSandbox joins root and requested into an absolute path and verifies
// the result stays within root. It rejects absolute paths and uses
// filepath.Clean to collapse ".." segments before the prefix check, preventing
// traversal attacks like "../../etc/passwd".
func resolveInSandbox(root, requested string) (string, error) {
	if filepath.IsAbs(requested) {
		return "", &ToolError{Kind: KindSandboxViolation, Err: fmt.Errorf("absolute paths not allowed: %s", requested)}
	}

	full := filepath.Clean(filepath.Join(root, requested))
	rootClean := filepath.Clean(root)

	if full != rootClean && !strings.HasPrefix(full, rootClean+string(filepath.Separator)) {
		return "", &ToolError{Kind: KindSandboxViolation, Err: fmt.Errorf("path escapes sandbox: %s", requested)}
	}

	return full, nil
}

// requireStringArg extracts a string value from args by key. It returns a
// KindInvalidArgument ToolError if the key is absent or the value is not a
// string, covering both missing and wrong-type cases with a single check.
func requireStringArg(args map[string]any, key string) (string, error) {
	val, ok := args[key].(string)
	if !ok {
		return "", &ToolError{Kind: KindInvalidArgument, Err: fmt.Errorf("missing or invalid %q argument", key)}
	}
	return val, nil
}

// Read implements Tool for reading files within a sandboxed directory. Root is
// the path to the sandbox directory; all file access is restricted to it.
type Read struct {
	Root string
}

func (r Read) Name() string {
	return "read"
}

func (r Read) Description() string {
	return "read a file"
}

func (r Read) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string"},
		},
		"required": []string{"path"},
	}
}

// Execute reads the file at args["path"] relative to Root and returns its
// contents. Sandbox violations and file errors are returned as ToolResult with
// IsError set; missing or invalid arguments are returned as Go errors.
func (r Read) Execute(args map[string]any) (core.ToolResult, error) {
	path, err := requireStringArg(args, "path")
	if err != nil {
		return core.ToolResult{}, err
	}

	resolvedPath, err := resolveInSandbox(r.Root, path)
	if err != nil {
		return errorResult(err.Error()), nil
	}

	contentBytes, err := os.ReadFile(resolvedPath)
	if err != nil {
		return fileErrorResult(err), nil
	}
	return core.ToolResult{Output: string(contentBytes)}, nil
}

// Write implements Tool for writing files within a sandboxed directory. Root is
// the path to the sandbox directory; all file access is restricted to it.
type Write struct {
	Root string
}

func (w Write) Name() string {
	return "write"
}

func (w Write) Description() string {
	return "write a file"
}

func (w Write) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string"},
			"content": map[string]any{"type": "string"},
		},
		"required": []string{"path", "content"},
	}
}

// Execute writes content from args["content"] to the file at args["path"]
// relative to Root. Sandbox violations and file errors are returned as
// ToolResult with IsError set; missing or invalid arguments are returned as Go
// errors.
func (w Write) Execute(args map[string]any) (core.ToolResult, error) {
	path, err := requireStringArg(args, "path")
	if err != nil {
		return core.ToolResult{}, err
	}

	resolvedPath, err := resolveInSandbox(w.Root, path)
	if err != nil {
		return errorResult(err.Error()), nil
	}

	content, err := requireStringArg(args, "content")
	if err != nil {
		return core.ToolResult{}, err
	}

	err = os.WriteFile(resolvedPath, []byte(content), 0o644)
	if err != nil {
		return fileErrorResult(err), nil
	}
	return core.ToolResult{}, nil
}
