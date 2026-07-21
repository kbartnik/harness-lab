package provider

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kbartnik/harness-lab/internal/core"
	"github.com/kbartnik/harness-lab/internal/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type capturedRequest struct {
	req  *http.Request
	body []byte
}

func setupTestServer(t *testing.T, resp anthropicResponse) *capturedRequest {
	t.Helper()

	captured := &capturedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.req = r
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}

		captured.body = body

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("failed to encode test response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	original := anthropicAPIURL
	anthropicAPIURL = server.URL
	t.Cleanup(func() { anthropicAPIURL = original })

	return captured
}

func TestAnthropicSendMessage(t *testing.T) {
	t.Run("sends message and parses text response", func(t *testing.T) {
		captured := setupTestServer(t, anthropicResponse{
			Content: []anthropicContentBlock{
				{Type: "text", Text: "here are the contents of foo.txt"},
			},
			StopReason: "end_turn",
			Usage:      anthropicUsage{InputTokens: 10, OutputTokens: 5},
		})

		result, err := AnthropicSendMessage([]core.Message{
			{Role: "user", Text: "read foo.txt"},
		}, nil, "fake-api-key")
		require.NoError(t, err)

		assert.Equal(t, "fake-api-key", captured.req.Header.Get("x-api-key"))
		assert.Equal(t, "2023-06-01", captured.req.Header.Get("anthropic-version"))
		assert.Equal(t, "application/json", captured.req.Header.Get("content-type"))
		assert.Equal(t, "here are the contents of foo.txt", result.Message.Text)
		assert.Equal(t, "end_turn", result.StopReason)
		assert.Equal(t, 10, result.InputTokens)
		assert.Equal(t, 5, result.OutputTokens)
	})

	t.Run("includes tools in request body", func(t *testing.T) {
		captured := setupTestServer(t, anthropicResponse{
			Content:    []anthropicContentBlock{{Type: "text", Text: "ok"}},
			StopReason: "end_turn",
		})

		_, err := AnthropicSendMessage([]core.Message{
			{Role: "user", Text: "read foo.txt"},
		}, []tool.Tool{tool.Read{}}, "fake-api-key")
		require.NoError(t, err)

		var sentReq anthropicRequest
		require.NoError(t, json.Unmarshal(captured.body, &sentReq))

		require.Len(t, sentReq.Tools, 1)
		assert.Equal(t, "read", sentReq.Tools[0].Name)
		assert.Equal(t, "read a file", sentReq.Tools[0].Description)

		wantSchemaBytes, err := json.Marshal(tool.Read{}.Schema())
		require.NoError(t, err)
		var wantSchema map[string]any
		require.NoError(t, json.Unmarshal(wantSchemaBytes, &wantSchema))

		assert.Equal(t, wantSchema, sentReq.Tools[0].InputSchema)
	})

	t.Run("returns error on non-2xx response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("invalid api key"))
		}))
		t.Cleanup(server.Close)

		original := anthropicAPIURL
		anthropicAPIURL = server.URL
		t.Cleanup(func() { anthropicAPIURL = original })

		_, err := AnthropicSendMessage([]core.Message{{Role: "user", Text: "hi"}}, nil, "bad-key")

		require.Error(t, err)
		var anthropicErr *AnthropicError
		require.ErrorAs(t, err, &anthropicErr)

		assert.Equal(t, http.StatusUnauthorized, anthropicErr.StatusCode)
		assert.Equal(t, "invalid api key", anthropicErr.Body)
		assert.Equal(t, "anthropic api error: status 401: invalid api key", err.Error())
	})
}

func TestAnthropicToolFromTool(t *testing.T) {
	result := anthropicToolFromTool(tool.Read{})

	assert.Equal(t, "read", result.Name)
	assert.Equal(t, "read a file", result.Description)
	assert.Equal(t, tool.Read{}.Schema(), result.InputSchema)
}
