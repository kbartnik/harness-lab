package provider

import (
	"testing"

	"github.com/kbartnik/harness-lab/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestResultFromResponse(t *testing.T) {
	t.Run("text-only response", func(t *testing.T) {
		resp := anthropicResponse{
			Content: []anthropicContentBlock{
				{Type: "text", Text: "here are the contents of foo.txt"},
			},
			StopReason: "end_turn",
			Usage: anthropicUsage{
				InputTokens:  100,
				OutputTokens: 20,
			},
		}

		result := resultFromResponse(resp)

		assert.Equal(t, "here are the contents of foo.txt", result.Message.Text)
		assert.Equal(t, "end_turn", result.StopReason)
		assert.Equal(t, 100, result.InputTokens)
		assert.Equal(t, 20, result.OutputTokens)
	})

	t.Run("tool-use response", func(t *testing.T) {
		resp := anthropicResponse{
			Content: []anthropicContentBlock{
				{
					Type: "text",
					Text: "here are the contents of foo.txt",
				},
				{
					Type:  "tool_use",
					ID:    "toolu01Xyz",
					Name:  "read",
					Input: map[string]any{"path": "foo.txt"},
				},
			},
		}

		result := resultFromResponse(resp)

		assert.Equal(t, 1, len(result.Message.ToolCalls))
		assert.Equal(t, "toolu01Xyz", result.Message.ToolCalls[0].ID)
		assert.Equal(t, "read", result.Message.ToolCalls[0].Name)
		assert.Equal(t, map[string]any{"path": "foo.txt"}, result.Message.ToolCalls[0].Args)
	})

	t.Run("skips unrecognized block types", func(t *testing.T) {
		//TODO: build an anthropicResponse whose content includes a
		//{"type": "thinking", ...}-shaped block alongside a normal
		//"text" block.
		resp := anthropicResponse{
			Content: []anthropicContentBlock{
				{
					Type: "thinking",
					Text: "internal reasoning that should be dropped.",
				},
				{
					Type: "text",
					Text: "here are the contents of foo.txt",
				},
			},
		}

		result := resultFromResponse(resp)

		assert.Equal(t, "here are the contents of foo.txt", result.Message.Text)
	})
}

func TestAnthropicMessageFromCore(t *testing.T) {
	t.Run("user message", func(t *testing.T) {
		msg := core.Message{
			Role: "user",
			Text: "Read foo.txt and tell me what it says.",
		}

		result := anthropicMessageFromCore(msg)

		assert.Equal(t, "user", result.Role)
		assert.Equal(t, []anthropicContentBlock{
			{Type: "text", Text: "Read foo.txt and tell me what it says."},
		}, result.Content)
	})

	t.Run("assistant message with test and tool call", func(t *testing.T) {
		msg := core.Message{
			Role: "assistant",
			Text: "I'll read that file for you.",
			ToolCalls: []core.ToolCall{
				{ID: "toolu01Xyz", Name: "read", Args: map[string]any{"path": "foo.txt"}},
			},
		}

		result := anthropicMessageFromCore(msg)

		assert.Equal(t, "assistant", result.Role)
		assert.Equal(t, []anthropicContentBlock{
			{Type: "text", Text: "I'll read that file for you."},
			{Type: "tool_use", ID: "toolu01Xyz", Name: "read", Input: map[string]any{"path": "foo.txt"}},
		}, result.Content)
	})

	t.Run("tool result message", func(t *testing.T) {
		msg := core.Message{
			Role:       "tool",
			ToolCallID: "toolu01Xyz",
			Text:       "the file's contents go here.",
		}

		result := anthropicMessageFromCore(msg)

		assert.Equal(t, "user", result.Role)
		assert.Equal(t, []anthropicContentBlock{
			{Type: "tool_result", ToolUseID: "toolu01Xyz", Content: "the file's contents go here."},
		}, result.Content)
	})
}
