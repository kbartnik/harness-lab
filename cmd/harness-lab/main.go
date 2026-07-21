package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/kbartnik/harness-lab/internal/core"
	"github.com/kbartnik/harness-lab/internal/loop"
	"github.com/kbartnik/harness-lab/internal/provider"
	"github.com/kbartnik/harness-lab/internal/tool"
)

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "ANTHROPIC_API_KEY not set")
		os.Exit(1)
	}

	if err := os.MkdirAll("sandbox", 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "cannot create sandbox:", err)
		os.Exit(1)
	}

	tools := []tool.Tool{
		tool.Read{Root: "sandbox"},
		tool.Write{Root: "sandbox"},
	}

	send := func(messages []core.Message, tools []tool.Tool) (provider.Result, error) {
		return provider.AnthropicSendMessage(messages, tools, apiKey)
	}

	var messages []core.Message

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")

	for scanner.Scan() {
		input := scanner.Text()
		if input == "" {
			fmt.Print("> ")
			continue
		}

		messages = append(messages, core.Message{Role: "user", Text: input})

		result, err := loop.Run(send, tools, messages)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			fmt.Print("> ")
			continue
		}
		messages = result
		fmt.Println(result[len(result)-1].Text)
		fmt.Print("> ")
	}
}
