//go:build integration

package provider

import (
	"os"
	"testing"

	"github.com/kbartnik/harness-lab/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicSendMessageIntegration(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping integration test")
	}

	messages := []core.Message{
		{Role: "user", Text: "reply with exactly the word hello and nothing else."},
	}

	result, err := AnthropicSendMessage(messages, nil, apiKey)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Message.Text)
	assert.NotEmpty(t, result.StopReason)
	assert.Greater(t, result.InputTokens, 0)
	assert.Greater(t, result.OutputTokens, 0)

	t.Logf("response %q, stop_reason: %s, tokens: in=%d out=%d",
		result.Message.Text, result.StopReason, result.InputTokens, result.OutputTokens)
}
