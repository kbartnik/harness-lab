package provider

import (
	"testing"

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
