package provider

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kbartnik/harness-lab/internal/core"
	"github.com/kbartnik/harness-lab/internal/tool"
)

var anthropicAPIURL = "https://api.anthropic.com/v1/messages"

const (
	anthropicModel     = "claude-sonnet-4-5"
	anthropicMaxTokens = 4096
)

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type anthropicMessage struct {
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

type anthropicContentBlock struct {
	Type      string         `json:"type"` // "text" | "tool_use" | "tool_result"
	Text      string         `json:"text,omitempty"`
	ID        string         `json:"id,omitempty"`          // tool_use_block's own id
	Name      string         `json:"name,omitempty"`        // tool_use_block's tool name
	Input     map[string]any `json:"input,omitempty"`       // tool_use_block's arguments
	ToolUseID string         `json:"tool_use_id,omitempty"` // tool_result references a tool_use id
	Content   string         `json:"content,omitempty"`     // tool_result's output
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicResponse struct {
	ID         string                  `json:"id"`
	Role       string                  `json:"role"`
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
	Usage      anthropicUsage          `json:"usage"`
}

type Result struct {
	Message      core.Message `json:"message"`
	StopReason   string       `json:"stop_reason"`
	InputTokens  int          `json:"input_tokens"`
	OutputTokens int          `json:"output_tokens"`
}

func resultFromResponse(resp anthropicResponse) Result {
	var text strings.Builder
	var toolCalls []core.ToolCall

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			text.WriteString(block.Text)
		case "tool_use":
			toolCalls = append(toolCalls, core.ToolCall{
				ID:   block.ID,
				Name: block.Name,
				Args: block.Input,
			})
		}
	}
	return Result{
		Message: core.Message{
			Role:      "assistant",
			Text:      text.String(),
			ToolCalls: toolCalls,
		},
		StopReason:   resp.StopReason,
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
	}
}

func anthropicMessageFromCore(m core.Message) anthropicMessage {
	var role string
	var content []anthropicContentBlock

	switch m.Role {
	case "user":
		role = "user"
		content = []anthropicContentBlock{
			{
				Type: "text",
				Text: m.Text,
			},
		}

	case "tool":
		role = "user"
		content = []anthropicContentBlock{
			{
				Type:      "tool_result",
				ToolUseID: m.ToolCallID,
				Content:   m.Text,
			},
		}

	case "assistant":
		role = "assistant"
		if m.Text != "" {
			content = append(content, anthropicContentBlock{Type: "text", Text: m.Text})
		}
		for _, call := range m.ToolCalls {
			content = append(content, anthropicContentBlock{Type: "tool_use", ID: call.ID, Name: call.Name, Input: call.Args})
		}
	}

	return anthropicMessage{
		Role:    role,
		Content: content,
	}
}

func AnthropicSendMessage(messages []core.Message, tools []tool.Tool, apiKey string) (Result, error) {
	anthropicMessages := make([]anthropicMessage, 0, len(messages))
	for _, m := range messages {
		anthropicMessages = append(anthropicMessages, anthropicMessageFromCore(m))
	}

	reqBody := anthropicRequest{
		Model:     anthropicModel,
		MaxTokens: anthropicMaxTokens,
		Messages:  anthropicMessages,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return Result{}, err
	}

	httpReq, err := http.NewRequest("POST", anthropicAPIURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return Result{}, err
	}

	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("content-type", "application/json")

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = httpResp.Body.Close() }()

	var anthropicResp anthropicResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&anthropicResp); err != nil {
		return Result{}, err
	}

	return resultFromResponse(anthropicResp), nil
}
