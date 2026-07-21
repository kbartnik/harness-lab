package loop

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kbartnik/harness-lab/internal/core"
	"github.com/kbartnik/harness-lab/internal/provider"
	"github.com/kbartnik/harness-lab/internal/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Run("no tool calls", func(t *testing.T) {
		calls := 0
		fakeSend := func(messages []core.Message, tools []tool.Tool) (provider.Result, error) {
			calls++
			return provider.Result{
				Message:    core.Message{Role: "assistant", Text: "hello"},
				StopReason: "end_turn",
			}, nil
		}

		result, err := Run(fakeSend, nil, []core.Message{{Role: "user", Text: "hi"}})

		require.NoError(t, err)
		assert.Equal(t, 1, calls)
		require.Len(t, result, 2)
		assert.Equal(t, "hello", result[1].Text)
	})

	t.Run("tool call then stop", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "foo.txt"), []byte("file contents"), 0o644))

		calls := 0
		fakeSend := func(messages []core.Message, tools []tool.Tool) (provider.Result, error) {
			calls++
			if calls == 1 {
				return provider.Result{
					Message: core.Message{
						Role: "assistant",
						ToolCalls: []core.ToolCall{
							{ID: "call1", Name: "read", Args: map[string]any{"path": "foo.txt"}},
						},
					},
					StopReason: "tool_use",
				}, nil
			}
			return provider.Result{
				Message:    core.Message{Role: "assistant", Text: "done"},
				StopReason: "end_turn",
			}, nil
		}

		tools := []tool.Tool{tool.Read{Root: dir}}

		result, err := Run(fakeSend, tools, []core.Message{{Role: "user", Text: "read foo.txt"}})

		require.NoError(t, err)
		assert.Equal(t, 2, calls)
		require.Len(t, result, 4)
		assert.Equal(t, "call1", result[1].ToolCalls[0].ID)
		assert.Equal(t, core.Role("tool"), result[2].Role)
		assert.Equal(t, "call1", result[2].ToolCallID)
		assert.Equal(t, "file contents", result[2].Text)
		assert.Equal(t, "done", result[3].Text)
	})

	t.Run("send error propagates", func(t *testing.T) {
		sentinelError := errors.New("send failed")

		fakeSend := func(messages []core.Message, tools []tool.Tool) (provider.Result, error) {
			return provider.Result{}, sentinelError
		}

		result, err := Run(fakeSend, nil, []core.Message{
			{Role: "user", Text: "hi"},
		})

		require.ErrorIs(t, err, sentinelError)
		assert.Nil(t, result)
	})

	t.Run("unknown tool", func(t *testing.T) {
		calls := 0
		fakeSend := func(messages []core.Message, tools []tool.Tool) (provider.Result, error) {
			calls++
			if calls == 1 {
				return provider.Result{
					Message: core.Message{
						Role: "assistant",
						ToolCalls: []core.ToolCall{
							{ID: "call1", Name: "bogus", Args: map[string]any{}},
						},
					},
					StopReason: "tool_use",
				}, nil
			}
			return provider.Result{
				Message:    core.Message{Role: "assistant", Text: "done"},
				StopReason: "end_turn",
			}, nil
		}

		result, err := Run(fakeSend, nil, []core.Message{
			{Role: "user", Text: "delete everything"},
		})

		require.NoError(t, err)
		assert.Equal(t, 2, calls)
		require.Len(t, result, 4)
		assert.Equal(t, core.Role("tool"), result[2].Role)
		assert.Equal(t, "call1", result[2].ToolCallID)
		assert.Equal(t, "unknown tool: bogus", result[2].Text)
		assert.Equal(t, "done", result[3].Text)
	})

	t.Run("does not mutate caller's backing array", func(t *testing.T) {
		initial := make([]core.Message, 1, 10)
		initial[0] = core.Message{Role: "user", Text: "hi"}
		backing := initial[:cap(initial)]

		fakeSend := func(messages []core.Message, tools []tool.Tool) (provider.Result, error) {
			return provider.Result{
				Message:    core.Message{Role: "assistant", Text: "hello"},
				StopReason: "end_turn",
			}, nil
		}

		_, err := Run(fakeSend, nil, initial)
		require.NoError(t, err)

		assert.Equal(t, core.Message{}, backing[1])
	})

	t.Run("max tool iterations exceeded", func(t *testing.T) {
		calls := 0
		fakeSend := func(messages []core.Message, tools []tool.Tool) (provider.Result, error) {
			calls++
			return provider.Result{
				Message: core.Message{
					Role: "assistant",
					ToolCalls: []core.ToolCall{
						{ID: "call1", Name: "bogus", Args: map[string]any{}},
					},
				},
				StopReason: "tool_use",
			}, nil
		}

		_, err := Run(fakeSend, nil, []core.Message{{Role: "user", Text: "loop forever"}})
		require.Error(t, err)

		assert.Equal(t, maxToolIterations+1, calls)
	})
}
