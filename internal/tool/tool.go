// Package tool contains the Tool interface and implementations
package tool

import "github.com/kbartnik/harness-lab/internal/core"

type Tool interface {
	Name() string
	Schema() map[string]any
	Execute(args map[string]any) (core.ToolResult, error)
}
