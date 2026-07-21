// Package core defines the shared types used across the provider, tool, and
// loop packages: Message and ToolCall/ToolResult for conversation wire format,
// and Turn for per-iteration instrumentation.
package core

import (
	"time"
)

// Role identifies the author of a Message: "user" for human turns,
// "assistant" for model turns, and "tool" for tool-result turns.
type Role string

// ToolCall represents a request from the model to invoke a named tool with
// the given arguments. ID is assigned by the provider and must be echoed back
// in the corresponding tool-result Message.
type ToolCall struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// ToolResult holds the outcome of executing a ToolCall. Output is returned to
// the model as the tool-result message content. IsError signals that the tool
// encountered a handled error (sandbox violation, file not found, etc.) — the
// model sees the error description and can reason about it.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Output     string `json:"output"`
	IsError    bool   `json:"is_error,omitempty"`
}

// Message is one turn in a conversation. Role determines how the message is
// interpreted: user messages carry Text, assistant messages carry Text and/or
// ToolCalls, and tool-result messages carry ToolCallID and Text (the output).
type Message struct {
	Role       Role       `json:"role"`
	Text       string     `json:"text,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // present on assistant messages that call tools
	ToolCallID string     `json:"tool_call_id,omitempty"` // present on tool-result messages
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
