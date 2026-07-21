package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypesJSONRoundTrip(t *testing.T) {
	t.Run("JSON round-trip preserves all Message fields", func(t *testing.T) {
		message := Message{
			Role: Role("assistant"),
			Text: "here are the contents of foo.txt",
			ToolCalls: []ToolCall{
				{
					ID:   "call_1",
					Name: "read",
					Args: map[string]any{"path": "foo.txt"},
				},
			},
		}

		bytes, err := json.Marshal(message)
		require.NoError(t, err)

		var result Message
		err = json.Unmarshal(bytes, &result)
		require.NoError(t, err)

		assert.Equal(t, message, result)
	})

	t.Run("JSON round-trip preserves all Turn fields", func(t *testing.T) {
		turn := Turn{
			Provider: "anthropic",
			Request: []Message{
				{
					Role: Role("user"),
					Text: "here is the contents of bar.txt",
					ToolCalls: []ToolCall{
						{
							ID:   "call_2",
							Name: "read",
							Args: map[string]any{"path": "bar.txt"},
						},
					},
				},
			},
			Response: Message{
				Role: Role("assistant"),
				Text: "let me see what is in bar.txt",
			},
			StopReason:   "got bored",
			StartedAt:    time.Now().Round(0),
			Duration:     250 * time.Millisecond,
			InputTokens:  100,
			OutputTokens: 100,
		}

		bytes, err := json.Marshal(turn)
		require.NoError(t, err)

		var result Turn
		err = json.Unmarshal(bytes, &result)
		require.NoError(t, err)

		assert.Equal(t, turn, result)
	})
}
