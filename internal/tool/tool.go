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
	Schema() map[string]any
	Execute(args map[string]any) (core.ToolResult, error)
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

type Read struct {
	Root string
}

func (r Read) Name() string {
	return "read"
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
	path := args["path"].(string)

	resolvedPath, err := resolveInSandbox(r.Root, path)
	if err != nil || len(resolvedPath) == 0 {
		return core.ToolResult{ToolCallID: "", Output: err.Error(), IsError: true}, err
	}

	contentBytes, err := os.ReadFile(resolvedPath)
	if err != nil || len(contentBytes) == 0 {
		return core.ToolResult{ToolCallID: "", Output: err.Error(), IsError: true}, err
	}

	return core.ToolResult{ToolCallID: "", Output: string(contentBytes), IsError: false}, nil
}
