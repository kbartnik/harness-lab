// Package tool contains the Tool interface and implementations
package tool

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kbartnik/harness-lab/internal/core"
)

type Tool interface {
	Name() string
	Description() string
	Schema() map[string]any
	Execute(args map[string]any) (core.ToolResult, error)
}

func errorResult(msg string) core.ToolResult {
	return core.ToolResult{IsError: true, Output: msg}
}

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

func requireStringArg(args map[string]any, key string) (string, error) {
	val, ok := args[key].(string)
	if !ok {
		return "", &ToolError{Kind: KindInvalidArgument, Err: fmt.Errorf("missing or invalid %q argument", key)}
	}
	return val, nil
}

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

func fileErrorResult(err error) core.ToolResult {
	if toolErr := classifyFileError(err); toolErr != nil {
		return errorResult(toolErr.Error())
	}
	return errorResult(err.Error())
}

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
