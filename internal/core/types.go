// Package core defines the shared wire and instrumentation types -
// messages, tool calls and results, and turns - used across the
// provider, tool, and loop packages.
package core

import "time"

type Role string // "user" | "assistant" | "tool"

type Message struct {
	Role       Role       `json:"role"`
	Text       string     `json:"text,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // present on assistant messages that call tools
	ToolCallID string     `json:"tool_call_id,omitempty"` // present on tool-result messages
}

type ToolCall struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Output     string `json:"output"`
	IsError    bool   `json:"is_error,omitempty"`
}

// Turn is the instrumentation envelope for one five-step loop iteration. It carries
// everything that isn't conversation content, keeping Message/ToolCall/ToolResult
// pure wire types with no telemetry riding along.
type Turn struct {
	Provider     string        `json:"provider"`               // "anthropic" | "openai"
	Request      []Message     `json:"request"`                // messages sent to the model this Turn
	Response     Message       `json:"response"`               // the assistant message returned
	ToolResults  []ToolResult  `json:"tool_results,omitempty"` // results of any tool calls executed this Turn
	StopReason   string        `json:"stop_reason"`            // normalized stop/finish reason
	StartedAt    time.Time     `json:"started_at"`
	Duration     time.Duration `json:"duration"`
	InputTokens  int           `json:"input_tokens"`
	OutputTokens int           `json:"output_tokens"`
}
